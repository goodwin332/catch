package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"catch/apps/api/internal/app/config"
	"catch/apps/api/internal/platform/db"
	"catch/apps/api/internal/platform/logger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config_load_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log := logger.New(cfg.Env)
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		log.Error("database_open_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.ApplyMigrations(ctx, pool, cfg.Database.MigrationsDir); err != nil {
		log.Error("migrations_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log.Info("migrations_applied")
}
