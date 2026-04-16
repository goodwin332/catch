package logger

import (
	"log/slog"
	"os"

	"catch/apps/api/internal/app/config"
)

func New(env config.Env) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if env == config.EnvLocal || env == config.EnvDevelopment {
		return slog.New(slog.NewTextHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
