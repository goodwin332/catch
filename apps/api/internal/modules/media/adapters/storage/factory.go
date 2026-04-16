package storage

import (
	"context"
	"fmt"

	"catch/apps/api/internal/app/config"
	"catch/apps/api/internal/modules/media/ports"
)

func New(ctx context.Context, cfg config.StorageConfig) (ports.Storage, error) {
	switch cfg.Provider {
	case "local":
		return NewLocalStorage(cfg.LocalPath), nil
	case "s3":
		return NewS3Storage(ctx, cfg)
	default:
		return nil, fmt.Errorf("unknown storage provider %q", cfg.Provider)
	}
}
