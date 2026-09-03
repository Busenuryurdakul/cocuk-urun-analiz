package mail

import (
	"context"
	"fmt"
	"net/smtp"
)

type SMTPService struct {
	host string
	port string
	from string
}

func NewSMTP(host, port, from string) *SMTPService {
	return &SMTPService{host: host, port: port, from: from}
}

func (s *SMTPService) Send(ctx context.Context, msg Message) error {
	_ = ctx
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	body := fmt.Sprintf("To: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", msg.To, msg.Subject, msg.Body)
	return smtp.SendMail(addr, nil, s.from, []string{msg.To}, []byte(body))
}
