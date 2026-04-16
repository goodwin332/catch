package mail

import (
	"context"
	"log/slog"
)

type Message struct {
	To      string
	Subject string
	Text    string
}

type Sender interface {
	Send(context.Context, Message) error
}

type NoopSender struct{}

func (NoopSender) Send(context.Context, Message) error {
	return nil
}

type LogSender struct {
	log *slog.Logger
}

func NewLogSender(log *slog.Logger) *LogSender {
	return &LogSender{log: log}
}

func (s *LogSender) Send(ctx context.Context, message Message) error {
	s.log.InfoContext(ctx, "mail_delivery_ready", slog.String("to", message.To), slog.String("subject", message.Subject))
	return nil
}
