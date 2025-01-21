package smtp

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/mail"
	"strconv"

	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/adapter"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/smtp"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/workflow"
)

//go:embed adapter
var adapterDir embed.FS

//go:embed adapter.json
var adapterDefinition string

func init() {
	adapter := adapter.RegisterAdapterByRaw([]byte(adapterDefinition))
	adapter.RegisterSpecsByDir(adapterDir)
	adapter.RegisterCredentialTestingFunc(testCredential)

	workflow.RegistryNodeMeta(&SendEmail{})
}

func testCredential(ctx context.Context, credentialType model.CredentialType, fields model.InputFields) (err error) {
	secureMethod := fields.GetString("security")
	host := fields.GetString("host")
	username := fields.GetString("username")
	password := fields.GetString("password")
	portStr := fields.GetString("port")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		err = fmt.Errorf("port must be a number: %w", err)
		return
	}

	authOpt := smtp.AuthOpt{
		SecureMethod: smtp.SecureMethod(secureMethod),
		Host:         host,
		Port:         port,
		Username:     username,
		Password:     password,
	}

	return smtp.TestAuth(ctx, authOpt)
}

type SendEmail struct {
	// The addresses of the receiver
	To []string `json:"to"`
	// the email subject
	Subject string `json:"subject"`
	// the message body of the email
	Body string `json:"body"`

	// Optional, attachments
	Attachments []Attachment `json:"attachments"`
	// Optional, carbon copy receivers of the email
	Cc []string `json:"cc"`
	// Is the body using HTML?
	HTMLBody bool `json:"htmlBody,string"`
}

type Attachment struct {
	// optional
	FileName string `json:"fileName"`
	// required, text or base64 text
	Content string `json:"content"`
}

func (a Attachment) MarshalJSON() ([]byte, error) {
	type plain Attachment

	const abstractLength = 10

	var (
		n       = len(a.Content)
		content string
	)
	if n == 0 {
		// relax
	} else if n > abstractLength {
		content = a.Content[:abstractLength] + "...(omitted)"
	} else {
		content = a.Content
	}

	omitted := plain{
		FileName: a.FileName,
		Content:  content,
	}

	return json.Marshal(omitted)
}

type Credential struct {
	Host       string            `json:"host"`
	Port       int               `json:"port,string"`
	Security   smtp.SecureMethod `json:"security"`
	Username   string            `json:"username"`
	Password   string            `json:"password"`
	SenderName string            `json:"senderName"`
}

func (s *SendEmail) UltrafoxNode() workflow.NodeMeta {
	spec := adapter.MustLookupSpec("ultrafox/smtp#sendMail")

	return workflow.NodeMeta{
		Class: spec.Class,
		New: func() workflow.Node {
			return &SendEmail{}
		},
		InputForm: spec.InputSchema,
	}
}

func (s *SendEmail) Run(c *workflow.NodeContext) (result any, err error) {
	var (
		ctx        = c.Context()
		credential Credential
	)
	err = c.GetAuthorizer().DecodeMeta(&credential)
	if err != nil {
		err = fmt.Errorf("decoding meta: %w", err)
		return
	}

	authOpt := smtp.AuthOpt{
		SecureMethod: credential.Security,
		Host:         credential.Host,
		Port:         credential.Port,
		Username:     credential.Username,
		Password:     credential.Password,
	}
	email := smtp.Mail{
		From: mail.Address{
			Name:    credential.SenderName,
			Address: credential.Username,
		},
		To:       removeEmptyItem(s.To),
		Subject:  s.Subject,
		Body:     []byte(s.Body),
		Cc:       removeEmptyItem(s.Cc),
		HTMLBody: s.HTMLBody,
	}

	for idx, attachment := range s.Attachments {
		if attachment.FileName == "" && attachment.Content == "" {
			// skip empty item
			continue
		}

		var content []byte
		content, badBase64 := base64.StdEncoding.DecodeString(attachment.Content)
		if badBase64 != nil {
			// content is plain text
			content = []byte(attachment.Content)
		}

		err = email.AddAttachment(smtp.Attachment{
			FileName: attachment.FileName,
			Content:  content,
		}, true)
		if err != nil {
			err = fmt.Errorf("adding attachment at index %d: %w", idx, err)
			return
		}
	}

	err = smtp.SendMail(ctx, authOpt, email)
	if err != nil {
		err = fmt.Errorf("sending mail: %w", err)
		return
	}

	result = map[string]any{
		"success": true,
	}

	return
}

func removeEmptyItem(s []string) (o []string) {
	if s == nil {
		return
	}

	o = make([]string, 0, len(s))
	for _, item := range s {
		if item == "" {
			continue
		}

		o = append(o, item)
	}

	return
}
