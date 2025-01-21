package smtp

import (
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func intactMail() Mail {
	return Mail{
		From: mail.Address{
			Name:    "John",
			Address: "john@42.io",
		},
		To:                     []string{"alice@wonderland.com", "bob@evil.com"},
		Subject:                "My chocolate factory is due open!",
		Body:                   []byte("Welcome!"),
		HTMLBody:               false,
		Cc:                     []string{"eve@eardropping.com"},
		Bcc:                    []string{"mom@home.nyc", "bob@evil.com"},
		ReplyTo:                "sale@johnschocolate.com",
		overallAttachmentBytes: 0,
		attachments:            nil,
		testSendTime:           time.Unix(1670403504, 0),
		testMultipartBoundary:  "myUniqueBoundary86sdeA1ad",
	}
}

func TestMail_Validate(t *testing.T) {
	noSender := intactMail()
	noSender.From.Address = ""
	noReceiver := intactMail()
	noReceiver.To = nil
	badReceiver := intactMail()
	badReceiver.To = []string{"a\r@b.com"}
	noSubject := intactMail()
	noSubject.Subject = ""
	noCC := intactMail()
	noCC.Cc = nil

	tests := []struct {
		name    string
		mail    Mail
		wantErr bool
	}{
		{
			name:    "intact",
			mail:    intactMail(),
			wantErr: false,
		},
		{
			name:    "no sender",
			mail:    noSender,
			wantErr: true,
		},
		{
			name:    "no receiver",
			mail:    noReceiver,
			wantErr: true,
		},
		{
			name:    "no subject",
			mail:    noSubject,
			wantErr: false,
		},
		{
			name:    "bad receiver",
			mail:    badReceiver,
			wantErr: true,
		},
		{
			name:    "no CC",
			mail:    noCC,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.mail.validate(); (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMail_Recipients(t *testing.T) {
	m := intactMail()
	expected := []string{"alice@wonderland.com", "bob@evil.com", "eve@eardropping.com", "mom@home.nyc"}
	require.ElementsMatch(t, m.Recipients(), expected)
}

func TestMail_Message_Text(t *testing.T) {
	m := intactMail()
	require.NoError(t, m.validate())
	msg, err := m.Message()
	require.NoError(t, err)

	expected := loadTestData("expected-mail-text.crlftxt")
	require.Equal(t, string(expected), string(msg))
}

func TestMail_Message_HTML(t *testing.T) {
	m := intactMail()
	m.HTMLBody = true
	require.NoError(t, m.validate())
	msg, err := m.Message()
	require.NoError(t, err)

	expected := loadTestData("expected-mail-html.crlftxt")
	require.Equal(t, string(expected), string(msg))
}

func loadTestData(name string) []byte {
	joined := filepath.Join("testdata", name)
	content, err := os.ReadFile(joined)
	if err != nil {
		err = fmt.Errorf("reading file %q: %w", joined, err)
		panic(err)
	}

	return content
}

func writeTestData(name string, content []byte) {
	joined := filepath.Join("testdata", name)

	err := os.WriteFile(joined, content, 0644)
	if err != nil {
		err = fmt.Errorf("writing file %q: %w", joined, err)
		panic(err)
	}
}

func TestMail_Message_Attachment(t *testing.T) {
	const body = `Four score and seven years ago our fathers brought forth, upon this continent, a new nation, conceived in liberty, and dedicated to the proposition that all men are created equal.`

	var (
		err    error
		assert = require.New(t)
	)

	m := intactMail()
	m.Body = []byte(body)

	err = m.AddAttachment(Attachment{Content: loadTestData("thumb-up.jpg")}, true)
	assert.NoError(err)
	err = m.AddAttachment(Attachment{Content: loadTestData("never-gonna-give-you-up.pdf")}, true)
	assert.NoError(err)

	require.NoError(t, m.validate())
	msg, err := m.Message()
	require.NoError(t, err)

	// To reproduce test data, run:
	// writeTestData("expected-mail-with-attachment.crlftxt", msg)
	expected := loadTestData("expected-mail-with-attachment.crlftxt")

	require.Equal(t, string(expected), string(msg))
}

type TestMailOpt struct {
	// The address of the sender, like alice@example.com
	From mail.Address
	// The addresses of the receiver
	To []string
	// Optional, carbon copy receivers of the email
	Cc []string
	// Optional, blind carbon copy receivers of the email
	Bcc []string
	// Optional, an email address that the reply should go to
	ReplyTo string
	// Is the body using HTML?
	HTMLBody bool
}

func TestMail_AddAttachment(t *testing.T) {
	var (
		invalidFile = []byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 0}
		textCorpse  = []byte("Four score and seven years ago our fathers brought forth, upon this continent, a new nation, conceived in liberty, and dedicated to the proposition that all men are created equal.")
	)

	var (
		err    error
		m      = Mail{}
		assert = require.New(t)
	)

	// reject missing filename
	err = m.AddAttachment(Attachment{}, false)
	require.Error(t, err)
	err = m.AddAttachment(Attachment{
		Content: textCorpse,
	}, false)
	require.Error(t, err)

	// infer filename: empty
	err = m.AddAttachment(Attachment{}, true)
	require.NoError(t, err)
	require.Len(t, m.attachments, 1)
	require.Equal(t, "attachment-1.txt", m.attachments[0].FileName)
	m.attachments = nil // reset

	// infer filename: invalid file
	err = m.AddAttachment(Attachment{
		Content: invalidFile,
	}, true)
	require.NoError(t, err)
	require.Len(t, m.attachments, 1)
	require.Equal(t, "attachment-1.bin", m.attachments[0].FileName)
	m.attachments = nil // reset

	// infer filename: text
	err = m.AddAttachment(Attachment{
		Content: textCorpse,
	}, true)
	require.NoError(t, err)
	require.Len(t, m.attachments, 1)
	require.Equal(t, "attachment-1.txt", m.attachments[0].FileName)
	m.attachments = nil // reset

	// infer filename: provided content-type
	err = m.AddAttachment(Attachment{
		Content:     nil,
		ContentType: "application/pdf",
	}, true)
	require.NoError(t, err)
	require.Len(t, m.attachments, 1)
	require.Equal(t, "attachment-1.pdf", m.attachments[0].FileName)
	m.attachments = nil // reset

	// byte size limit
	err = m.AddAttachment(Attachment{
		Content:     make([]byte, maxAttachmentBytes+1),
		ContentType: "application/pdf",
	}, true)
	require.Error(t, err)
	m.attachments = nil // reset

	// attachment num limit
	var k Mail
	for i := 0; i < maxAttachments; i++ {
		err = k.AddAttachment(Attachment{}, true)
		assert.NoError(err)
	}
	err = k.AddAttachment(Attachment{}, true)
	assert.Error(err)
}
