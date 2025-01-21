package smtp

import (
	"bytes"
	"context"
	"fmt"
	"net/mail"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/require"
)

func mailForTest(opt TestMailOpt, subjectSuffix string) Mail {
	const scaffold = `{{- if .HTMLBody -}} <html lang="en-US"><body style="white-space: pre-line"> {{- end -}}
Hello! 
This is a test mail initiated by UltraFox to test the SMTP support.

The mail is stated to be from: {{ .From.Name }} <{{ .From.Address }}>.
{{if .ReplyTo }}The mail has a reply-to address: {{ .ReplyTo }} {{- end}}

To:
{{- range $index, $item := .To }}
* {{ $item }}
{{- else }}
<empty>
{{- end }}

Cc:
{{- range $index, $item := .Cc }}
* {{ $item }}
{{- else }}
<empty>
{{- end }}

Bcc:
{{- range $index, $item := .Bcc }}
* {{ $item }}
{{- else }}
<empty>
{{- end }}

HTML Test(will be raw HTML if not using HTMLBody mode):

<h1>A HTML H1 Title</h1>
<p>Here is a test link: <a href="https://example.com">Click me!</a></p>

Thank you and have a nice day!
{{- if .HTMLBody -}} </body></html> {{- end -}}
`

	var body bytes.Buffer
	err := template.Must(template.New("").Parse(scaffold)).Execute(&body, opt)
	if err != nil {
		err = fmt.Errorf("rendering mail body: %w", err)
		panic(err)
	}

	return Mail{
		From: opt.From,
		To:   opt.To,
		// Use timestamp to bypass Mail provider's deduplication logic
		Subject:  fmt.Sprintf("[%s] A SMTP Test Mail from UltraFox - %s", time.Now().Format(time.RFC3339), subjectSuffix),
		Body:     body.Bytes(),
		Cc:       opt.Cc,
		Bcc:      opt.Bcc,
		ReplyTo:  opt.ReplyTo,
		HTMLBody: opt.HTMLBody,
	}
}

const (
	r0 = "znli@gitlab.cn"
	r1 = "fuzhang@gitlab.cn"
	r2 = "ruili@gitlab.cn"
	r3 = "guoxd@gitlab.cn"
	r4 = "wyang@gitlab.cn"
	r5 = "weiduan@gitlab.cn"
	r6 = "jingliu@gitlab.cn"
)

func TestSendMail(t *testing.T) {
	if os.Getenv("TEST_SMTP") != "1" {
		t.Skip()
	}

	var (
		username = os.Getenv("SMTP_USER")
		password = os.Getenv("SMTP_PASSWORD")
		assert   = require.New(t)
	)

	assert.NotEmpty(username, "SMTP user name is required")
	assert.NotEmpty(password, "SMTP password is required")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// QQ: https://service.mail.qq.com/cgi-bin/help?id=28&no=167&subtype=1
	// Aliyun: https://help.aliyun.com/document_detail/36576.html
	// Gmail: https://support.google.com/a/answer/176600
	var authSSL AuthOpt
	switch vendor := strings.ToLower(os.Getenv("SMTP_VENDOR")); vendor {
	case "qq":
		authSSL = AuthOpt{
			SecureMethod: TLS,
			Host:         "smtp.qq.com",
			Port:         587,
			Username:     username,
			Password:     password,
		}
	case "aliyun":
		authSSL = AuthOpt{
			SecureMethod: SSL,
			Host:         "smtp.qiye.aliyun.com",
			Port:         465,
			Username:     username,
			Password:     password,
		}
	default:
		t.Errorf("unexpected SMTP vendor %q", vendor)
	}

	tests := []struct {
		name    string
		mail    TestMailOpt
		wantErr bool
	}{
		{
			name: "text",
			mail: TestMailOpt{
				From: mail.Address{
					Name:    "UltraFox Tester",
					Address: username,
				},
				To:      []string{r4, r1, r6},
				Cc:      []string{r2, r5},
				Bcc:     []string{r3, r0},
				ReplyTo: r5,
			},
			wantErr: false,
		},
		{
			name: "HTML",
			mail: TestMailOpt{
				From: mail.Address{
					Name:    "UltraFox Tester",
					Address: username,
				},
				To:       []string{r0},
				HTMLBody: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SendMail(ctx, authSSL, mailForTest(tt.mail, tt.name)); (err != nil) != tt.wantErr {
				t.Errorf("SendMail() via SSL error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSendMail_WithAttachments(t *testing.T) {
	if os.Getenv("TEST_SMTP") != "1" {
		t.Skip()
	}

	var (
		username = os.Getenv("SMTP_USER")
		password = os.Getenv("SMTP_PASSWORD")
		assert   = require.New(t)
	)

	assert.NotEmpty(username, "SMTP user name is required")
	assert.NotEmpty(password, "SMTP password is required")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// QQ: https://service.mail.qq.com/cgi-bin/help?id=28&no=167&subtype=1
	// Aliyun: https://help.aliyun.com/document_detail/36576.html
	// Gmail: https://support.google.com/a/answer/176600
	var authSSL AuthOpt
	switch vendor := strings.ToLower(os.Getenv("SMTP_VENDOR")); vendor {
	case "qq":
		authSSL = AuthOpt{
			SecureMethod: TLS,
			Host:         "smtp.qq.com",
			Port:         587,
			Username:     username,
			Password:     password,
		}
	case "aliyun":
		authSSL = AuthOpt{
			SecureMethod: SSL,
			Host:         "smtp.qiye.aliyun.com",
			Port:         465,
			Username:     username,
			Password:     password,
		}
	default:
		t.Errorf("unexpected SMTP vendor %q", vendor)
	}

	tests := []struct {
		name    string
		mail    TestMailOpt
		wantErr bool
	}{
		{
			name: "text with attachments",
			mail: TestMailOpt{
				From: mail.Address{
					Name:    "UltraFox Tester",
					Address: username,
				},
				To: []string{r0},
			},
			wantErr: false,
		},
		{
			name: "HTML with attachments",
			mail: TestMailOpt{
				From: mail.Address{
					Name:    "UltraFox Tester",
					Address: username,
				},
				To:       []string{r0},
				HTMLBody: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				err    error
				assert = require.New(t)
			)

			m := mailForTest(tt.mail, tt.name)
			err = m.AddAttachment(Attachment{Content: loadTestData("thumb-up.jpg")}, true)
			assert.NoError(err)
			err = m.AddAttachment(Attachment{Content: loadTestData("never-gonna-give-you-up.pdf")}, true)
			assert.NoError(err)
			// test if UTF-8 filename is working
			err = m.AddAttachment(Attachment{FileName: "gitlab之歌.txt", Content: loadTestData("the-gitlab-song.txt")}, true)
			assert.NoError(err)

			if err := SendMail(ctx, authSSL, m); (err != nil) != tt.wantErr {
				t.Errorf("SendMail() via SSL error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateEmailAddress(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "empty",
			addr:    "",
			wantErr: true,
		},
		{
			name:    "bad",
			addr:    "abcgsae",
			wantErr: true,
		},
		{
			name:    "good",
			addr:    "alice@example.com",
			wantErr: false,
		},
		{
			name:    "good with name",
			addr:    "Alice <alice@example.com>",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateEmailAddress(tt.addr); (err != nil) != tt.wantErr {
				t.Errorf("validateEmailAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
