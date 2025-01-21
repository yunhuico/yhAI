package smtp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

const (
	maxAttachments     = 10
	maxAttachmentBytes = 10 << 20
)

// Mail a mail to be sent
type Mail struct {
	// The address of the sender, like alice@example.com
	//
	// Address in AuthOpt is used when leaving blank.
	From mail.Address
	// The addresses of the receiver
	To []string
	// Optional, the email subject
	Subject string
	// Optional, the message body of the email
	Body []byte
	// Is the body using HTML?
	HTMLBody bool

	// Optional, carbon copy receivers of the email
	Cc []string
	// Optional, blind carbon copy receivers of the email
	Bcc []string
	// Optional, an email address that the reply should go to
	ReplyTo string

	overallAttachmentBytes int
	attachments            []Attachment

	// for testing
	testSendTime          time.Time
	testMultipartBoundary string
}

// Attachment mail attachment
type Attachment struct {
	// file name including extension, like hello.txt
	FileName string
	// MIME content type of the attachment,
	// will be automatically inferred per the content if not provided
	ContentType string
	// file content in binary format
	Content []byte
}

func (a Attachment) generateMIMEHeader() (h textproto.MIMEHeader) {
	h = make(textproto.MIMEHeader)
	h.Set("Content-Type", a.ContentType)
	h.Set("Content-Transfer-Encoding", "base64")
	h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, a.FileName))

	return
}

func (a Attachment) writeContent(writer io.Writer) (err error) {
	w := NewBase64Encoder(writer)
	_, err = w.Write(a.Content)
	if err != nil {
		err = fmt.Errorf("encoding base64: %w", err)
		return
	}

	err = w.Close()
	if err != nil {
		err = fmt.Errorf("finishing base64 encoding: %w", err)
		return
	}

	_, err = w.Write([]byte("\r\n"))
	if err != nil {
		err = fmt.Errorf("finishing with newline: %w", err)
		return
	}

	return
}

// AddAttachment add a new attachment to the Mail.
// When autoFileName is true, a filename including extension is generated when FileName is missing,
// otherwise an error returns.
func (m *Mail) AddAttachment(attachment Attachment, autoFileName bool) (err error) {
	if attachment.FileName == "" && !autoFileName {
		err = errors.New("fileName is missing and autoFileName is off")
		return
	}
	if strings.ContainsAny(attachment.FileName, "\r\n\"") {
		err = fmt.Errorf("filename can not contain linefeed, return, or quote, got %q", attachment.FileName)
		return
	}

	if attachment.ContentType == "" {
		attachment.ContentType = http.DetectContentType(attachment.Content)
	} else {
		_, _, err = mime.ParseMediaType(attachment.ContentType)
		if err != nil {
			err = fmt.Errorf("checking ContentType %q: %w", attachment.ContentType, err)
			return
		}
	}
	if attachment.FileName == "" {
		var exts []string
		exts, err = mime.ExtensionsByType(attachment.ContentType)
		if err != nil {
			err = fmt.Errorf("infering file extension from content type %q: %w", attachment.ContentType, err)
			return
		}

		attachment.FileName = fmt.Sprintf("attachment-%d%s", len(m.attachments)+1, famousExtension(exts))
	}

	m.attachments = append(m.attachments, attachment)
	m.overallAttachmentBytes += len(attachment.Content)

	if len(m.attachments) > maxAttachments {
		err = fmt.Errorf("too many attachments, exceding limit %d", maxAttachments)
		return
	}
	if m.overallAttachmentBytes > maxAttachmentBytes {
		err = fmt.Errorf("overall attachment size %d bytes exceding limit %d bytes", m.overallAttachmentBytes, maxAttachmentBytes)
		return
	}

	return
}

func famousExtension(exts []string) string {
	if len(exts) == 0 {
		return ".bin"
	}

	for _, ext := range exts {
		switch ext {
		case ".txt", ".html", ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
			".jpeg", ".jpg", ".png", ".svg", ".bmp", ".gif", ".ico", ".ps", ".psd", ".tiff", ".tif",
			".avi", ".mp4", ".mov", ".wmv",
			".7z", ".rar", ".tar.gz", ".zip":
			return ext
		}
	}

	return exts[0]
}

func (m Mail) validate() (err error) {
	err = validateEmailAddress(m.From.Address)
	if err != nil {
		err = fmt.Errorf("validating sender address: %w", err)
		return
	}
	err = validSMTPHeaderLine(m.From.String())
	if err != nil {
		err = fmt.Errorf("validating field From: %w", err)
		return
	}
	if len(m.To) == 0 {
		err = errors.New("field To is required")
		return
	}
	err = validateEmailAddress(m.To...)
	if err != nil {
		err = fmt.Errorf("validating receiver address: %w", err)
		return
	}
	err = validSMTPHeaderLine(m.To...)
	if err != nil {
		err = fmt.Errorf("validating field To: %w", err)
		return
	}
	err = validSMTPHeaderLine(m.Subject)
	if err != nil {
		err = fmt.Errorf("validating field Subject: %w", err)
		return
	}
	err = validSMTPHeaderLine(m.Cc...)
	if err != nil {
		err = fmt.Errorf("validating field Cc: %w", err)
		return
	}
	err = validateEmailAddress(m.Cc...)
	if err != nil {
		err = fmt.Errorf("validating Cc address: %w", err)
		return
	}
	err = validSMTPHeaderLine(m.Bcc...)
	if err != nil {
		err = fmt.Errorf("validating Bcc address: %w", err)
		return
	}
	err = validateEmailAddress(m.Bcc...)
	if err != nil {
		err = fmt.Errorf("validating Bcc address: %w", err)
		return
	}
	if m.ReplyTo != "" {
		err = validSMTPHeaderLine(m.ReplyTo)
		if err != nil {
			err = fmt.Errorf("validating field ReplyTo: %w", err)
			return
		}
		err = validateEmailAddress(m.ReplyTo)
		if err != nil {
			err = fmt.Errorf("validating ReplyTo address: %w", err)
			return
		}
	}

	// After base64, the mail will inflate by a factor of 1.5.
	// Normally a smtp mail can not exceed 20 MB.
	//
	// For simplicity, we ignore the size of headers.
	const maxMailSizeBytes = 12 << 20
	if mailSize := len(m.Body) + m.overallAttachmentBytes; mailSize > maxMailSizeBytes {
		err = fmt.Errorf("mail size %d bytes exceding limit %d bytes", mailSize, maxMailSizeBytes)
		return
	}

	return
}

func (m Mail) Recipients() []string {
	set := make(map[string]struct{}, len(m.To)+len(m.Cc)+len(m.Bcc))
	for _, item := range m.To {
		set[item] = struct{}{}
	}
	for _, item := range m.Cc {
		set[item] = struct{}{}
	}
	for _, item := range m.Bcc {
		set[item] = struct{}{}
	}

	mails := make([]string, 0, len(set))
	for k := range set {
		mails = append(mails, k)
	}

	return mails
}

func (m Mail) bodyHeader() (h textproto.MIMEHeader) {
	h = make(textproto.MIMEHeader)
	if m.HTMLBody {
		h.Set("Content-type", "text/html; charset=UTF-8")
	} else {
		h.Set("Content-type", "text/plain; charset=UTF-8")
	}

	h.Set("Content-Transfer-Encoding", "base64")

	return h
}

func (m Mail) writeBody(writer io.Writer) (err error) {
	if len(m.Body) == 0 {
		return
	}

	w := NewBase64Encoder(writer)
	_, err = w.Write(m.Body)
	if err != nil {
		err = fmt.Errorf("encoding base64: %w", err)
		return
	}

	err = w.Close()
	if err != nil {
		err = fmt.Errorf("finishing base64 encoding: %w", err)
		return
	}

	_, err = w.Write([]byte("\r\n"))
	if err != nil {
		err = fmt.Errorf("finishing with newline: %w", err)
		return
	}

	return
}

// Message produce the email data to be consumed by smtp
func (m Mail) Message() (msg []byte, err error) {
	var buf bytes.Buffer

	_, _ = fmt.Fprintf(&buf, "From: %s\r\n", m.From.String())
	_, _ = fmt.Fprintf(&buf, "To: %s\r\n", strings.Join(m.To, ", "))
	if m.Subject != "" {
		_, _ = fmt.Fprintf(&buf, "Subject: %s\r\n", m.Subject)
	}
	if len(m.Cc) > 0 {
		_, _ = fmt.Fprintf(&buf, "Cc: %s\r\n", strings.Join(m.Cc, ", "))
	}
	if m.ReplyTo != "" {
		_, _ = fmt.Fprintf(&buf, "Reply-To: %s\r\n", m.ReplyTo)
	}
	_, _ = buf.WriteString("MIME-Version: 1.0\r\n")

	// RFC5332 Origination Date Field
	// https://www.rfc-editor.org/rfc/rfc5322.html#page-22:~:text=Format%20%20%20%20%20%20%20%20%20%20%20%20%20October%202008-,3.6.1.%20%20The%20Origination%20Date%20Field,-The%20origination%20date
	if m.testSendTime.IsZero() {
		_, _ = fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	} else {
		_, _ = fmt.Fprintf(&buf, "Date: %s\r\n", m.testSendTime.UTC().Format(time.RFC1123Z))
	}

	if len(m.attachments) == 0 {
		// no attachments, go with the easy way
		if m.HTMLBody {
			_, _ = buf.WriteString("Content-type: text/html; charset=UTF-8\r\n")
		} else {
			_, _ = buf.WriteString("Content-type: text/plain; charset=UTF-8\r\n")
		}
		_, _ = buf.WriteString("Content-Transfer-Encoding: base64\r\n")

		// divider of header and body
		_, _ = buf.WriteString("\r\n")

		_ = m.writeBody(&buf)

		msg = buf.Bytes()
		return
	}

	// we have attachments, content will be arranged in the following MIME multiparts:
	//
	// * mixed
	//   * html / text
	//   * attachment 1
	//   * attachment 2
	//   * ...
	//   * attachment N
	// ref: https://stackoverflow.com/a/23853079
	parts := multipart.NewWriter(&buf)
	if m.testMultipartBoundary != "" {
		err = parts.SetBoundary(m.testMultipartBoundary)
		if err != nil {
			err = fmt.Errorf("setting multipartBoundary %q: %w", m.testMultipartBoundary, err)
			return
		}
	}

	_, _ = fmt.Fprintf(&buf, "Content-Type: multipart/mixed; boundary=%s\r\n", parts.Boundary())

	// divider of header and body
	_, _ = buf.WriteString("\r\n")

	part, _ := parts.CreatePart(m.bodyHeader())
	_ = m.writeBody(part)

	for _, attachment := range m.attachments {
		// divider between previous part and this one
		_, _ = buf.WriteString("\r\n")
		part, _ = parts.CreatePart(attachment.generateMIMEHeader())
		_ = attachment.writeContent(part)
	}

	parts.Close()

	msg = buf.Bytes()
	return
}
