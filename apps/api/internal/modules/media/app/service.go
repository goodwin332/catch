package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/media/app/dto"
	"catch/apps/api/internal/modules/media/domain"
	"catch/apps/api/internal/modules/media/ports"
	httpx "catch/apps/api/internal/platform/http"
)

type Service struct {
	repo           ports.Repository
	storage        ports.Storage
	publicBaseURL  string
	maxUploadBytes int64
	now            func() time.Time
}

func NewService(repo ports.Repository, storage ports.Storage, publicBaseURL string, maxUploadBytes int64) *Service {
	return &Service{repo: repo, storage: storage, publicBaseURL: strings.TrimRight(publicBaseURL, "/"), maxUploadBytes: maxUploadBytes, now: time.Now}
}

func (s *Service) Upload(ctx context.Context, actor accessdomain.Principal, originalName string, data []byte) (dto.FileResponse, error) {
	if !actor.CanCreateArticle() {
		return dto.FileResponse{}, httpx.Forbidden("Недостаточно прав для загрузки файлов")
	}
	if len(data) == 0 {
		return dto.FileResponse{}, mapMediaError(domain.ErrInvalidFile)
	}
	if int64(len(data)) > s.maxUploadBytes {
		return dto.FileResponse{}, mapMediaError(domain.ErrFileTooLarge)
	}
	mimeType := http.DetectContentType(data)
	if !allowedMimeType(mimeType) {
		return dto.FileResponse{}, mapMediaError(domain.ErrInvalidFile)
	}
	width, height, err := imageDimensions(mimeType, data)
	if err != nil {
		return dto.FileResponse{}, mapMediaError(err)
	}
	cleanName := cleanFileName(originalName)
	storageKey := randomStorageKey(mimeType)
	if err := s.storage.Save(ctx, storageKey, data); err != nil {
		return dto.FileResponse{}, err
	}
	file, err := s.repo.Create(ctx, ports.CreateFileInput{
		UploaderID:   actor.UserID,
		StorageKey:   storageKey,
		OriginalName: cleanName,
		MimeType:     mimeType,
		SizeBytes:    int64(len(data)),
		Width:        width,
		Height:       height,
	})
	if err != nil {
		_ = s.storage.Delete(ctx, storageKey)
		return dto.FileResponse{}, err
	}
	return s.mapFile(file), nil
}

func (s *Service) Get(ctx context.Context, fileID string) (dto.FileResponse, error) {
	file, err := s.repo.FindReady(ctx, fileID)
	if err != nil {
		return dto.FileResponse{}, mapMediaError(err)
	}
	return s.mapFile(file), nil
}

func (s *Service) Content(ctx context.Context, fileID string) (domain.File, []byte, error) {
	file, err := s.repo.FindReady(ctx, fileID)
	if err != nil {
		return domain.File{}, nil, mapMediaError(err)
	}
	data, err := s.storage.Open(ctx, file.StorageKey)
	if err != nil {
		return domain.File{}, nil, err
	}
	return file, data, nil
}

func (s *Service) CleanupUnreferenced(ctx context.Context, olderThan time.Duration, limit int) (int, error) {
	if olderThan < 0 {
		olderThan = 0
	}
	candidates, err := s.repo.ListUnreferencedReady(ctx, ports.CleanupCandidatesInput{
		Before: s.now().Add(-olderThan),
		Limit:  normalizeCleanupLimit(limit),
	})
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, file := range candidates {
		if err := s.storage.Delete(ctx, file.StorageKey); err != nil {
			return deleted, err
		}
		if err := s.repo.MarkDeleted(ctx, file.ID); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func (s *Service) mapFile(file domain.File) dto.FileResponse {
	return dto.FileResponse{
		ID:           file.ID,
		OriginalName: file.OriginalName,
		MimeType:     file.MimeType,
		SizeBytes:    file.SizeBytes,
		Width:        file.Width,
		Height:       file.Height,
		URL:          s.publicBaseURL + "/" + file.ID + "/content",
		CreatedAt:    file.CreatedAt.Format(time.RFC3339),
	}
}

func normalizeCleanupLimit(limit int) int {
	if limit <= 0 {
		return 100
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

func mapMediaError(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidFile):
		return httpx.ValidationError("Файл указан некорректно", map[string]any{"file": "invalid"})
	case errors.Is(err, domain.ErrFileTooLarge):
		return httpx.NewError(http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "Файл слишком большой")
	case errors.Is(err, domain.ErrFileNotFound):
		return httpx.NewError(http.StatusNotFound, httpx.CodeNotFound, "Файл не найден")
	default:
		return err
	}
}

func allowedMimeType(mimeType string) bool {
	switch mimeType {
	case "image/jpeg", "image/png", "image/webp", "image/gif", "application/pdf":
		return true
	default:
		return false
	}
}

func imageDimensions(mimeType string, data []byte) (*int, *int, error) {
	switch mimeType {
	case "image/jpeg", "image/png", "image/gif":
	default:
		return nil, nil, nil
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, nil, domain.ErrInvalidFile
	}
	if cfg.Width > 12000 || cfg.Height > 12000 {
		return nil, nil, domain.ErrInvalidFile
	}
	return &cfg.Width, &cfg.Height, nil
}

func cleanFileName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == "/" || base == "" {
		return "upload"
	}
	return base
}

func randomStorageKey(mimeType string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}
	exts, _ := mime.ExtensionsByType(mimeType)
	ext := ""
	if len(exts) > 0 {
		ext = exts[0]
	}
	return hex.EncodeToString(b[:]) + ext
}
