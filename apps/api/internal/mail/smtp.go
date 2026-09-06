package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
)

type SMTPConfig struct {
	Host     string
	Port     string
	From     string
	User     string
	Password string
	UseTLS   bool
}

type SMTPService struct {
	cfg SMTPConfig
}

func NewSMTP(cfg SMTPConfig) *SMTPService {
	return &SMTPService{cfg: cfg}
}

func NewSMTPFromHostPort(host, port, from string) *SMTPService {
	return NewSMTP(SMTPConfig{Host: host, Port: port, From: from})
}

func (s *SMTPService) Send(ctx context.Context, msg Message) error {
	_ = ctx
	addr := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	envelopeFrom, headerFrom := parseFrom(s.cfg.From)
	body := buildMIME(headerFrom, msg)
	if s.cfg.UseTLS && s.cfg.User != "" {
		return s.sendTLS(addr, envelopeFrom, msg.To, body)
	}
	return smtp.SendMail(addr, nil, envelopeFrom, []string{msg.To}, []byte(body))
}

func parseFrom(raw string) (envelope, header string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, raw
	}
	if addr, err := mail.ParseAddress(raw); err == nil {
		return addr.Address, raw
	}
	return raw, raw
}

func (s *SMTPService) sendTLS(addr, from, to, body string) error {
	host := s.cfg.Host
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(smtp.PlainAuth("", s.cfg.User, s.cfg.Password, host)); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(body)); err != nil {
		return err
	}
	return w.Close()
}

func buildMIME(from string, msg Message) string {
	fromLine := ""
	if from != "" {
		fromLine = fmt.Sprintf("From: %s\r\n", from)
	}
	if msg.HTMLBody != "" {
		boundary := "miyuna-mail-boundary"
		var b strings.Builder
		fmt.Fprintf(&b, "%sTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n", fromLine, msg.To, msg.Subject, boundary)
		fmt.Fprintf(&b, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", boundary, msg.Body)
		fmt.Fprintf(&b, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n", boundary, msg.HTMLBody)
		fmt.Fprintf(&b, "--%s--\r\n", boundary)
		return b.String()
	}
	return fmt.Sprintf("%sTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", fromLine, msg.To, msg.Subject, msg.Body)
}
