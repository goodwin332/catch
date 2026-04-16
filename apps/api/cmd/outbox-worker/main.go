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
	"catch/apps/api/internal/platform/mail"
	"catch/apps/api/internal/platform/outbox"
	"catch/apps/api/internal/platform/search"
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

	workerID, err := os.Hostname()
	if err != nil || workerID == "" {
		workerID = "catch-outbox-worker"
	}
	worker := outbox.NewWorker(pool, outbox.NewNotificationHandler(pool, search.NewArticleIndexer(cfg.Search), mail.NewSender(cfg.Mail, log), log), log, workerID)
	if err := worker.Run(ctx); err != nil && err != context.Canceled {
		log.Error("outbox_worker_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
