package mail

import (
	"log/slog"

	"catch/apps/api/internal/app/config"
)

func NewSender(cfg config.MailConfig, log *slog.Logger) Sender {
	switch cfg.Provider {
	case "smtp":
		return NewSMTPSender(SMTPConfig{
			From:     cfg.From,
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
		})
	case "disabled":
		return NoopSender{}
	default:
		return NewLogSender(log)
	}
}
