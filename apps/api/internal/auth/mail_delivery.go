package auth

import (
	"context"
	"errors"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mail"
)

type immediateMailer interface {
	SendImmediate(context.Context, mail.Message) error
}

func (s *Service) deliverCriticalMail(ctx context.Context, msg mail.Message) error {
	if s.Mail == nil {
		return errors.New("mail service is not configured")
	}
	if cm, ok := s.Mail.(immediateMailer); ok {
		return cm.SendImmediate(ctx, msg)
	}
	return s.Mail.Send(ctx, msg)
}
