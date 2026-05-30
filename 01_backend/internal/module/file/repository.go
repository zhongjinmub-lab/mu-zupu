package file

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return Repository{DB: db}
}

func (r Repository) NewID() string {
	return uuid.NewString()
}

func (r Repository) Create(ctx context.Context, in CreateFileInput) (File, error) {
	const q = `
INSERT INTO files(id, tenant_id, bucket, object_key, filename, mime_type, size_bytes, checksum, created_by)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, NULLIF($8, ''), $9)
RETURNING id::text, tenant_id::text, bucket, object_key, filename, COALESCE(mime_type, ''), size_bytes,
          COALESCE(checksum, ''), created_by::text, created_at`
	item := File{}
	err := r.DB.QueryRow(ctx, q,
		r.NewID(),
		in.TenantID,
		in.Bucket,
		in.ObjectKey,
		in.Filename,
		in.MimeType,
		in.SizeBytes,
		in.Checksum,
		in.CreatedBy,
	).Scan(&item.ID, &item.TenantID, &item.Bucket, &item.ObjectKey, &item.Filename, &item.MimeType, &item.SizeBytes, &item.Checksum, &item.CreatedBy, &item.CreatedAt)
	return item, err
}

func (r Repository) List(ctx context.Context, tenantID string, limit int) ([]File, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	const q = `
SELECT id::text, tenant_id::text, bucket, object_key, filename, COALESCE(mime_type, ''), size_bytes,
       COALESCE(checksum, ''), created_by::text, created_at
FROM files
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $2`
	rows, err := r.DB.Query(ctx, q, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]File, 0)
	for rows.Next() {
		var item File
		if err := rows.Scan(&item.ID, &item.TenantID, &item.Bucket, &item.ObjectKey, &item.Filename, &item.MimeType, &item.SizeBytes, &item.Checksum, &item.CreatedBy, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) Get(ctx context.Context, tenantID, fileID string) (File, error) {
	const q = `
SELECT id::text, tenant_id::text, bucket, object_key, filename, COALESCE(mime_type, ''), size_bytes,
       COALESCE(checksum, ''), created_by::text, created_at
FROM files
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	var item File
	err := r.DB.QueryRow(ctx, q, fileID, tenantID).Scan(&item.ID, &item.TenantID, &item.Bucket, &item.ObjectKey, &item.Filename, &item.MimeType, &item.SizeBytes, &item.Checksum, &item.CreatedBy, &item.CreatedAt)
	return item, err
}

func IsNotFound(err error) bool {
	return err == pgx.ErrNoRows
}
