package smtp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
)

type SecureMethod string

const (
	// SSL normally listens at port 465
	//
	// The client connects and immediately establishes a TLS handshake, just like with HTTPS.
	// This kind of connection is often incorrectly named "SSL" by email providers.
	//
	// Ref: https://gist.github.com/chrisgillis/10888032?permalink_comment_id=2553469#gistcomment-2553469
	SSL SecureMethod = "SSL"
	// TLS normally listens at port 587
	//
	// Port 587 is a cleartext SMTP port.
	// This means that the client connects, introduces itself in plaintext,
	// then sends a STARTTLS command and TLS handshake happens.
	// This kind of connection is incorrectly named "TLS" by email providers.
	TLS SecureMethod = "TLS"
	// None uses no transport security during authentication,
	// which exposes username and password in plain text.
	//
	// None is highly dangerous and NOT recommended.
	None SecureMethod = "None"
)

type AuthOpt struct {
	// TLS, SSL or None?
	SecureMethod SecureMethod
	// SMTP host
	Host string
	// SMTP port like 465 or 587
	Port int
	// Usually the same as the sender address
	Username string
	// Usually an "APP secret"
	Password string
}

func (o AuthOpt) validate() (err error) {
	if o.Host == "" {
		err = errors.New("host is missing")
		return
	}
	if o.Port <= 0 {
		err = fmt.Errorf("port is invalid, got %d", o.Port)
		return
	}
	if o.Username == "" {
		err = errors.New("username is missing")
		return
	}
	if o.Password == "" {
		err = errors.New("password is missing")
		return
	}

	return
}

func validSMTPHeaderLine(line ...string) (err error) {
	for _, l := range line {
		if strings.ContainsAny(l, "\r\n") {
			err = fmt.Errorf("a line must not contain CR or LF, got %q", l)
			return
		}
	}

	return
}

func validateEmailAddress(addr ...string) (err error) {
	for _, item := range addr {
		_, err = mail.ParseAddress(item)
		if err != nil {
			err = fmt.Errorf("parsing email address %q: %w", item, err)
			return
		}
	}

	return
}

func TestAuth(ctx context.Context, auth AuthOpt) (err error) {
	var conn net.Conn
	conn, err = dialSMTPServer(ctx, auth)
	if err != nil {
		err = fmt.Errorf("dialing SMTPServer: %w", err)
		return
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, auth.Host)
	if err != nil {
		err = fmt.Errorf("init SMTP client: %w", err)
		return
	}
	defer c.Close()

	err = authAccount(c, auth)
	if err != nil {
		err = fmt.Errorf("auth mail account: %w", err)
		return
	}
	return
}

func dialSMTPServer(ctx context.Context, opt AuthOpt) (conn net.Conn, err error) {
	var (
		method = opt.SecureMethod
		host   = opt.Host
		port   = opt.Port
	)
	switch method {
	case SSL:
		dialer := tls.Dialer{
			Config: &tls.Config{ServerName: host},
		}
		conn, err = dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			err = fmt.Errorf("dialing SSL: %w", err)
			return
		}
	case TLS, None:
		dialer := net.Dialer{}
		conn, err = dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			err = fmt.Errorf("dialing TCP: %w", err)
			return
		}
	default:
		err = fmt.Errorf("unexpected SMTP SecureMethod %q", method)
		return
	}

	deadline, _ := ctx.Deadline()
	err = conn.SetDeadline(deadline)
	if err != nil {
		err = fmt.Errorf("setting connection deadline: %w", err)
		return
	}
	return
}

func authAccount(c *smtp.Client, auth AuthOpt) (err error) {
	if auth.SecureMethod == TLS {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			err = errors.New("server does not support STARTTLS")
			return
		}
		err = c.StartTLS(&tls.Config{ServerName: auth.Host})
		if err != nil {
			err = fmt.Errorf("SMTP StartTLS: %w", err)
			return
		}
	}

	if ok, _ := c.Extension("AUTH"); !ok {
		err = errors.New("smtp: server doesn't support AUTH")
		return
	}
	err = c.Auth(PlainAuth("", auth.Username, auth.Password, auth.Host, auth.SecureMethod == None))
	if err != nil {
		err = fmt.Errorf("SMTP authing: %w", err)
		return
	}
	return
}

func SendMail(ctx context.Context, auth AuthOpt, mail Mail) (err error) {
	err = auth.validate()
	if err != nil {
		err = fmt.Errorf("validating auth: %w", err)
		return
	}

	if mail.From.Address == "" && mail.From.Name == "" {
		mail.From.Address = auth.Username
	}

	err = mail.validate()
	if err != nil {
		err = fmt.Errorf("validating mail: %w", err)
		return
	}

	var conn net.Conn
	conn, err = dialSMTPServer(ctx, auth)
	if err != nil {
		err = fmt.Errorf("dialing SMTPServer: %w", err)
		return
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, auth.Host)
	if err != nil {
		err = fmt.Errorf("init SMTP client: %w", err)
		return
	}
	defer c.Close()

	err = authAccount(c, auth)
	if err != nil {
		err = fmt.Errorf("auth mail account: %w", err)
		return
	}

	err = c.Mail(mail.From.Address)
	if err != nil {
		err = fmt.Errorf("assigning sender %q: %w", mail.From.Address, err)
		return
	}
	for _, addr := range mail.Recipients() {
		err = c.Rcpt(addr)
		if err != nil {
			err = fmt.Errorf("assign recipient %q: %w", addr, err)
			return
		}
	}

	w, err := c.Data()
	if err != nil {
		err = fmt.Errorf("calling SMTP data method: %w", err)
		return
	}
	msg, err := mail.Message()
	if err != nil {
		err = fmt.Errorf("forging mail message: %w", err)
		return
	}
	_, err = w.Write(msg)
	if err != nil {
		err = fmt.Errorf("writing mail header and message: %w", err)
		return
	}
	err = w.Close()
	if err != nil {
		err = fmt.Errorf("closing message writer: %w", err)
		return
	}

	err = c.Quit()
	if err != nil {
		err = fmt.Errorf("SMTP quiting: %w", err)
		return
	}

	return
}

type plainAuth struct {
	identity, username, password string
	host                         string
	allowInsecureTransport       bool
}

// PlainAuth returns an Auth that implements the PLAIN authentication
// mechanism as defined in RFC 4616. The returned Auth uses the given
// username and password to authenticate to host and act as identity.
// Usually identity should be the empty string, to act as username.
//
// This implementation varies from smtp.PlainAuth that it does not require
// TLS when allowInsecureTransport is true.
func PlainAuth(identity, username, password, host string, allowInsecureTransport bool) smtp.Auth {
	return &plainAuth{identity, username, password, host, allowInsecureTransport}
}

func isLocalhost(name string) bool {
	return name == "localhost" || name == "127.0.0.1" || name == "::1"
}

func (a *plainAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	// Must have TLS, or else localhost server.
	// Note: If TLS is not true, then we can't trust ANYTHING in ServerInfo.
	// In particular, it doesn't matter if the server advertises PLAIN auth.
	// That might just be the attacker saying
	// "it's ok, you can trust me with your password."
	if !a.allowInsecureTransport && !server.TLS && !isLocalhost(server.Name) {
		return "", nil, errors.New("unencrypted connection")
	}
	if server.Name != a.host {
		return "", nil, errors.New("wrong host name")
	}
	resp := []byte(a.identity + "\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}

func (a *plainAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		// We've already sent everything.
		return nil, errors.New("unexpected server challenge")
	}
	return nil, nil
}
