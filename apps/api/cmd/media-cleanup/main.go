package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"catch/apps/api/internal/app/composition"
	"catch/apps/api/internal/app/config"
	"catch/apps/api/internal/platform/logger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	olderThan := flag.Duration("older-than", 24*time.Hour, "delete unreferenced media files older than this duration")
	limit := flag.Int("limit", 100, "maximum files to delete in one run")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config_load_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	log := logger.New(cfg.Env)
	container, err := composition.New(ctx, cfg, log)
	if err != nil {
		log.Error("app_bootstrap_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer container.Close()

	deleted, err := container.Media.CleanupUnreferenced(ctx, *olderThan, *limit)
	if err != nil {
		log.Error("media_cleanup_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	log.Info("media_cleanup_completed", slog.Int("deleted", deleted))
}
