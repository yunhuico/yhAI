package smtp

import (
	"context"
	"fmt"
	"net/mail"
)

type Sender struct {
	auth AuthOpt
	from mail.Address
}

type SenderConfig struct {
	// credential to log in the smtp service
	Auth AuthOpt `comment:"credential to log in the smtp service"`
	// optional, sender address and name, represented in the from field of the email
	From mail.Address `comment:"optional, sender address and name, represented in the from field of the email"`
}

func NewSender(cfg SenderConfig) (s *Sender, err error) {
	err = cfg.Auth.validate()
	if err != nil {
		err = fmt.Errorf("validating auth: %w", err)
		return
	}

	s = &Sender{
		auth: cfg.Auth,
		from: cfg.From,
	}
	return
}

func (s *Sender) SendMail(ctx context.Context, mail Mail) (err error) {
	if mail.From.Address == "" && mail.From.Name == "" {
		mail.From = s.from
	}
	if mail.From.Name == "" {
		mail.From.Name = s.from.Name
	}

	err = SendMail(ctx, s.auth, mail)
	return
}
