package storage

import (
	"context"
	"os"
	"path/filepath"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) *LocalStorage {
	return &LocalStorage{basePath: basePath}
}

func (s *LocalStorage) Save(_ context.Context, key string, data []byte) error {
	path := filepath.Join(s.basePath, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	err := os.Remove(filepath.Join(s.basePath, key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *LocalStorage) Open(_ context.Context, key string) ([]byte, error) {
	return os.ReadFile(filepath.Join(s.basePath, key))
}
