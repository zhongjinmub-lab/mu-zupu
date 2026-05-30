package license

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return Repository{DB: db}
}

func (r Repository) List(ctx context.Context, tenantID string) ([]License, error) {
	const q = `
SELECT id::text, tenant_id::text, license_no, license_type, status, subject, limits,
       COALESCE(public_key_id, ''), COALESCE(signature, ''), issued_at,
       activated_at, revoked_at, expired_at, created_at, updated_at
FROM licenses
WHERE tenant_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC`
	rows, err := r.DB.Query(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]License, 0)
	for rows.Next() {
		item, err := scanLicense(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) Get(ctx context.Context, tenantID, licenseID string) (License, error) {
	const q = `
SELECT id::text, tenant_id::text, license_no, license_type, status, subject, limits,
       COALESCE(public_key_id, ''), COALESCE(signature, ''), issued_at,
       activated_at, revoked_at, expired_at, created_at, updated_at
FROM licenses
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	item, err := scanLicense(r.DB.QueryRow(ctx, q, licenseID, tenantID))
	if err == pgx.ErrNoRows {
		return License{}, ErrNotFound
	}
	return item, err
}

func (r Repository) Create(ctx context.Context, tenantID string, req CreateLicenseRequest) (License, error) {
	req.Normalize()
	if req.LicenseNo == "" {
		req.LicenseNo = newLicenseNo()
	}
	subject, err := jsonObject(req.Subject)
	if err != nil {
		return License{}, err
	}
	limits, err := jsonObject(req.Limits)
	if err != nil {
		return License{}, err
	}
	const q = `
INSERT INTO licenses(tenant_id, license_no, license_type, subject, limits, public_key_id, signature, expired_at)
VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, NULLIF($6, ''), NULLIF($7, ''), $8)
RETURNING id::text, tenant_id::text, license_no, license_type, status, subject, limits,
          COALESCE(public_key_id, ''), COALESCE(signature, ''), issued_at,
          activated_at, revoked_at, expired_at, created_at, updated_at`
	return scanLicense(r.DB.QueryRow(ctx, q, tenantID, req.LicenseNo, req.LicenseType, subject, limits, req.PublicKeyID, req.Signature, req.ExpiredAt))
}

func (r Repository) Activate(ctx context.Context, tenantID, licenseID string) (License, error) {
	const q = `
UPDATE licenses
SET status = 'active',
    activated_at = COALESCE(activated_at, now()),
    revoked_at = NULL,
    updated_at = now()
WHERE id = $1
  AND tenant_id = $2
  AND deleted_at IS NULL
  AND status IN ('inactive', 'active')
  AND (expired_at IS NULL OR expired_at > now())
RETURNING id::text, tenant_id::text, license_no, license_type, status, subject, limits,
          COALESCE(public_key_id, ''), COALESCE(signature, ''), issued_at,
          activated_at, revoked_at, expired_at, created_at, updated_at`
	item, err := scanLicense(r.DB.QueryRow(ctx, q, licenseID, tenantID))
	if err == pgx.ErrNoRows {
		return License{}, ErrNotFound
	}
	return item, err
}

func (r Repository) Revoke(ctx context.Context, tenantID, licenseID string) (License, error) {
	const q = `
UPDATE licenses
SET status = 'revoked',
    revoked_at = now(),
    updated_at = now()
WHERE id = $1
  AND tenant_id = $2
  AND deleted_at IS NULL
  AND status <> 'revoked'
RETURNING id::text, tenant_id::text, license_no, license_type, status, subject, limits,
          COALESCE(public_key_id, ''), COALESCE(signature, ''), issued_at,
          activated_at, revoked_at, expired_at, created_at, updated_at`
	item, err := scanLicense(r.DB.QueryRow(ctx, q, licenseID, tenantID))
	if err == pgx.ErrNoRows {
		return License{}, ErrNotFound
	}
	return item, err
}

type licenseScanner interface {
	Scan(dest ...any) error
}

func scanLicense(row licenseScanner) (License, error) {
	var item License
	err := row.Scan(
		&item.ID, &item.TenantID, &item.LicenseNo, &item.LicenseType, &item.Status,
		&item.Subject, &item.Limits, &item.PublicKeyID, &item.Signature, &item.IssuedAt,
		&item.ActivatedAt, &item.RevokedAt, &item.ExpiredAt, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func jsonObject(v map[string]any) (string, error) {
	if v == nil {
		return "{}", nil
	}
	b, err := json.Marshal(v)
	return string(b), err
}

func newLicenseNo() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "LIC-" + time.Now().UTC().Format("20060102150405")
	}
	return "LIC-" + hex.EncodeToString(b[:])
}

var ErrNotFound = pgx.ErrNoRows
