package mail

import (
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strings"
)

type SMTPConfig struct {
	From     string
	Host     string
	Port     int
	Username string
	Password string
}

type SMTPSender struct {
	cfg SMTPConfig
}

func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Send(ctx context.Context, message Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	from, err := mail.ParseAddress(s.cfg.From)
	if err != nil {
		return err
	}
	to, err := mail.ParseAddress(message.To)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	return smtp.SendMail(addr, auth, from.Address, []string{to.Address}, buildMessage(from.String(), to.String(), message))
}

func buildMessage(from, to string, message Message) []byte {
	var buffer bytes.Buffer
	subject := mime.QEncoding.Encode("utf-8", message.Subject)
	headers := []string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
	}
	buffer.WriteString(strings.Join(headers, "\r\n"))
	buffer.WriteString("\r\n\r\n")
	buffer.WriteString(message.Text)
	buffer.WriteString("\r\n")
	return buffer.Bytes()
}
