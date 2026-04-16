package postgres

import (
	"context"
	"errors"

	"catch/apps/api/internal/modules/media/domain"
	"catch/apps/api/internal/modules/media/ports"
	"catch/apps/api/internal/platform/db"

	"github.com/jackc/pgx/v5"
)

type Repository struct {
	tx *db.TxManager
}

func NewRepository(tx *db.TxManager) *Repository {
	return &Repository{tx: tx}
}

func (r *Repository) Create(ctx context.Context, input ports.CreateFileInput) (domain.File, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		insert into media_files (uploader_id, storage_key, original_name, mime_type, size_bytes, width, height)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id::text, uploader_id::text, storage_key, original_name, mime_type, size_bytes, width, height, status, created_at
	`, input.UploaderID, input.StorageKey, input.OriginalName, input.MimeType, input.SizeBytes, input.Width, input.Height)
	return scanFile(row)
}

func (r *Repository) FindReady(ctx context.Context, fileID string) (domain.File, error) {
	row := r.tx.Querier(ctx).QueryRow(ctx, `
		select id::text, uploader_id::text, storage_key, original_name, mime_type, size_bytes, width, height, status, created_at
		from media_files
		where id = $1 and status = 'ready'
	`, fileID)
	return scanFile(row)
}

func (r *Repository) ListUnreferencedReady(ctx context.Context, input ports.CleanupCandidatesInput) ([]domain.File, error) {
	rows, err := r.tx.Querier(ctx).Query(ctx, `
		select id::text, uploader_id::text, storage_key, original_name, mime_type, size_bytes, width, height, status, created_at
		from media_files f
		where f.status = 'ready'
			and f.created_at < $1
			and not exists (
				select 1
				from article_revision_media_files rm
				where rm.file_id = f.id
			)
		order by f.created_at asc, f.id asc
		limit $2
	`, input.Before, input.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]domain.File, 0)
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

func (r *Repository) MarkDeleted(ctx context.Context, fileID string) error {
	_, err := r.tx.Querier(ctx).Exec(ctx, `
		update media_files
		set status = 'deleted'
		where id = $1 and status = 'ready'
	`, fileID)
	return err
}

type fileScanner interface {
	Scan(...any) error
}

func scanFile(row fileScanner) (domain.File, error) {
	var file domain.File
	if err := row.Scan(
		&file.ID,
		&file.UploaderID,
		&file.StorageKey,
		&file.OriginalName,
		&file.MimeType,
		&file.SizeBytes,
		&file.Width,
		&file.Height,
		&file.Status,
		&file.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.File{}, domain.ErrFileNotFound
		}
		return domain.File{}, err
	}
	return file, nil
}
