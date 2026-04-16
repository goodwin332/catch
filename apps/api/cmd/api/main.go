package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"catch/apps/api/internal/app/bootstrap"
	"catch/apps/api/internal/app/composition"
	"catch/apps/api/internal/app/config"
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
	container, err := composition.New(ctx, cfg, log)
	if err != nil {
		log.Error("app_bootstrap_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer container.Close()

	server := bootstrap.NewServer(cfg, log, bootstrap.NewRouter(container))
	if err := server.Run(ctx); err != nil {
		log.Error("api_server_failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
