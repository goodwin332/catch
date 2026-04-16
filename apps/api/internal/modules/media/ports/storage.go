package ports

import "context"

type Storage interface {
	Save(context.Context, string, []byte) error
	Delete(context.Context, string) error
	Open(context.Context, string) ([]byte, error)
}
