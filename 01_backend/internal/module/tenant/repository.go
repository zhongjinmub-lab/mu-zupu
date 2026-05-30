package tenant

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
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

func (r Repository) Create(ctx context.Context, userID, name, code string) (Tenant, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return Tenant{}, err
	}
	defer tx.Rollback(ctx)

	const createTenant = `
INSERT INTO tenants(name, code)
VALUES ($1, $2)
RETURNING id::text, name, code, status, created_at, updated_at`
	var t Tenant
	if err := tx.QueryRow(ctx, createTenant, name, code).Scan(&t.ID, &t.Name, &t.Code, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return Tenant{}, err
	}

	const createMember = `
INSERT INTO tenant_members(tenant_id, user_id, role_code, status)
VALUES ($1, $2, 'owner', 'active')`
	if _, err := tx.Exec(ctx, createMember, t.ID, userID); err != nil {
		return Tenant{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Tenant{}, err
	}
	t.RoleCode = "owner"
	return t, nil
}

func (r Repository) ListByUser(ctx context.Context, userID string) ([]Tenant, error) {
	const q = `
SELECT t.id::text, t.name, t.code, t.status, tm.role_code, t.created_at, t.updated_at
FROM tenants t
JOIN tenant_members tm ON tm.tenant_id = t.id
WHERE tm.user_id = $1
  AND tm.status = 'active'
  AND t.deleted_at IS NULL
  AND tm.deleted_at IS NULL
ORDER BY t.created_at DESC`
	rows, err := r.DB.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Tenant, 0)
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Code, &t.Status, &t.RoleCode, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r Repository) GetForUser(ctx context.Context, tenantID, userID string) (Tenant, error) {
	const q = `
SELECT t.id::text, t.name, t.code, t.status, tm.role_code, t.created_at, t.updated_at
FROM tenants t
JOIN tenant_members tm ON tm.tenant_id = t.id
WHERE t.id = $1
  AND tm.user_id = $2
  AND tm.status = 'active'
  AND t.status = 'active'
  AND t.deleted_at IS NULL
  AND tm.deleted_at IS NULL`
	var t Tenant
	err := r.DB.QueryRow(ctx, q, tenantID, userID).Scan(&t.ID, &t.Name, &t.Code, &t.Status, &t.RoleCode, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r Repository) ListMembers(ctx context.Context, tenantID string, limit int) ([]Member, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	const q = `
SELECT tm.id::text, tm.tenant_id::text, tm.user_id::text, u.email::text, COALESCE(u.nickname, ''),
       tm.role_code, tm.status, tm.created_at, tm.updated_at
FROM tenant_members tm
JOIN users u ON u.id = tm.user_id
WHERE tm.tenant_id = $1
  AND tm.deleted_at IS NULL
  AND u.deleted_at IS NULL
ORDER BY tm.created_at ASC
LIMIT $2`
	rows, err := r.DB.Query(ctx, q, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Member, 0)
	for rows.Next() {
		item, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) AddMember(ctx context.Context, tenantID string, req AddMemberRequest) (Member, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return Member{}, err
	}
	const q = `
WITH target_user AS (
    SELECT id
    FROM users
    WHERE email = $2 AND status = 'active' AND deleted_at IS NULL
)
INSERT INTO tenant_members(tenant_id, user_id, role_code, status)
SELECT $1, id, $3, 'active' FROM target_user
ON CONFLICT (tenant_id, user_id) DO UPDATE
SET role_code = EXCLUDED.role_code,
    status = 'active',
    deleted_at = NULL,
    updated_at = now()
RETURNING id::text, tenant_id::text, user_id::text,
          (SELECT email::text FROM users WHERE users.id = tenant_members.user_id),
          COALESCE((SELECT nickname FROM users WHERE users.id = tenant_members.user_id), ''),
          role_code, status, created_at, updated_at`
	item, err := scanMember(r.DB.QueryRow(ctx, q, tenantID, req.Email, req.RoleCode))
	if err == pgx.ErrNoRows {
		return Member{}, ErrUserNotFound
	}
	return item, err
}

func (r Repository) UpdateMemberRole(ctx context.Context, tenantID, memberID string, req UpdateMemberRoleRequest) (Member, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return Member{}, err
	}
	const q = `
UPDATE tenant_members
SET role_code = $3,
    updated_at = now()
WHERE id = $2 AND tenant_id = $1 AND deleted_at IS NULL AND status = 'active'
RETURNING id::text, tenant_id::text, user_id::text,
          (SELECT email::text FROM users WHERE users.id = tenant_members.user_id),
          COALESCE((SELECT nickname FROM users WHERE users.id = tenant_members.user_id), ''),
          role_code, status, created_at, updated_at`
	item, err := scanMember(r.DB.QueryRow(ctx, q, tenantID, memberID, req.RoleCode))
	if err == pgx.ErrNoRows {
		return Member{}, ErrMemberNotFound
	}
	return item, err
}

func (r Repository) RemoveMember(ctx context.Context, tenantID, memberID string) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	const currentQ = `
SELECT role_code
FROM tenant_members
WHERE id = $2 AND tenant_id = $1 AND deleted_at IS NULL AND status = 'active'
FOR UPDATE`
	var role string
	if err := tx.QueryRow(ctx, currentQ, tenantID, memberID).Scan(&role); err != nil {
		if err == pgx.ErrNoRows {
			return ErrMemberNotFound
		}
		return err
	}
	if role == "owner" {
		return ErrCannotRemoveOwner
	}
	const q = `
UPDATE tenant_members
SET status = 'inactive',
    deleted_at = now(),
    updated_at = now()
WHERE id = $2 AND tenant_id = $1 AND deleted_at IS NULL`
	if _, err := tx.Exec(ctx, q, tenantID, memberID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r Repository) InsertAuditLog(ctx context.Context, tenantID, actorUserID, action, resourceType, resourceID, ip, userAgent string, metadata map[string]any) error {
	meta, err := jsonObject(metadata)
	if err != nil {
		return err
	}
	const q = `
INSERT INTO audit_logs(tenant_id, actor_user_id, action, resource_type, resource_id, ip, user_agent, metadata)
VALUES (NULLIF($1, '')::uuid, NULLIF($2, '')::uuid, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), $8::jsonb)`
	_, err = r.DB.Exec(ctx, q, tenantID, actorUserID, action, resourceType, resourceID, ip, userAgent, meta)
	return err
}

func (r Repository) ListAuditLogs(ctx context.Context, q AuditLogQuery) ([]AuditLog, string, error) {
	return r.listAuditLogs(ctx, q, true)
}
func (r Repository) ExportAuditLogs(ctx context.Context, q AuditLogQuery) ([]AuditLog, error) {
	q.CursorRaw = ""
	q.Cursor = AuditLogCursor{}
	if q.Limit <= 0 || q.Limit > 1000 {
		q.Limit = 1000
	}
	items, _, err := r.listAuditLogs(ctx, q, false)
	return items, err
}

func (r Repository) listAuditLogs(ctx context.Context, q AuditLogQuery, withCursor bool) ([]AuditLog, string, error) {
	q.Normalize()
	if err := q.Validate(); err != nil {
		return nil, "", err
	}
	args := []any{q.TenantID}
	where := `WHERE tenant_id = $1`
	addFilter := func(sql string, value any) {
		args = append(args, value)
		where += ` AND ` + sql + `$` + strconv.Itoa(len(args))
	}
	if q.Action != "" {
		addFilter(`action = `, q.Action)
	}
	if q.ResourceType != "" {
		addFilter(`resource_type = `, q.ResourceType)
	}
	if q.ActorUserID != "" {
		args = append(args, q.ActorUserID)
		where += ` AND actor_user_id = $` + strconv.Itoa(len(args)) + `::uuid`
	}
	if !q.From.IsZero() {
		addFilter(`created_at >= `, q.From)
	}
	if !q.To.IsZero() {
		addFilter(`created_at <= `, q.To)
	}
	if withCursor && !q.Cursor.CreatedAt.IsZero() {
		args = append(args, q.Cursor.CreatedAt, q.Cursor.ID)
		where += ` AND (created_at, id) < ($` + strconv.Itoa(len(args)-1) + `, $` + strconv.Itoa(len(args)) + `::uuid)`
	}
	limit := q.Limit
	if withCursor {
		limit++
	}
	args = append(args, limit)
	limitPlaceholder := "$" + strconv.Itoa(len(args))
	sql := `
SELECT id::text, COALESCE(tenant_id::text, ''), COALESCE(actor_user_id::text, ''), action,
       COALESCE(resource_type, ''), COALESCE(resource_id, ''), COALESCE(ip, ''), COALESCE(user_agent, ''),
       metadata, created_at
FROM audit_logs
` + where + `
ORDER BY created_at DESC, id DESC
LIMIT ` + limitPlaceholder
	rows, err := r.DB.Query(ctx, sql, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	out := make([]AuditLog, 0)
	for rows.Next() {
		var item AuditLog
		if err := rows.Scan(&item.ID, &item.TenantID, &item.ActorUserID, &item.Action, &item.ResourceType, &item.ResourceID, &item.IP, &item.UserAgent, &item.Metadata, &item.CreatedAt); err != nil {
			return nil, "", err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	nextCursor := ""
	if withCursor && len(out) > q.Limit {
		nextCursor = EncodeAuditLogCursor(out[q.Limit-1])
		out = out[:q.Limit]
	}
	return out, nextCursor, nil
}

func (r Repository) CreateInvitation(ctx context.Context, tenantID, invitedBy string, req CreateInvitationRequest) (Invitation, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return Invitation{}, err
	}
	token, tokenHash, err := newInvitationToken()
	if err != nil {
		return Invitation{}, err
	}
	const q = `
INSERT INTO tenant_invitations(tenant_id, email, role_code, token_hash, invited_by, expired_at)
VALUES ($1, $2, $3, $4, $5, now() + ($6::int || ' hours')::interval)
RETURNING id::text, tenant_id::text, email::text, role_code, status, COALESCE(invited_by::text, ''),
          COALESCE(accepted_by::text, ''), accepted_at, revoked_at, expired_at, created_at, updated_at`
	item, err := scanInvitation(r.DB.QueryRow(ctx, q, tenantID, req.Email, req.RoleCode, tokenHash, invitedBy, req.TTLHours))
	if err != nil {
		return Invitation{}, err
	}
	item.Token = token
	return item, nil
}

func (r Repository) ListInvitations(ctx context.Context, tenantID, status string, limit int) ([]Invitation, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	args := []any{tenantID, limit}
	where := `WHERE tenant_id = $1 AND deleted_at IS NULL`
	limitPlaceholder := "$2"
	if status != "" {
		args = []any{tenantID, status, limit}
		where += ` AND status = $2`
		limitPlaceholder = "$3"
	}
	q := `
SELECT id::text, tenant_id::text, email::text, role_code, status, COALESCE(invited_by::text, ''),
       COALESCE(accepted_by::text, ''), accepted_at, revoked_at, expired_at, created_at, updated_at
FROM tenant_invitations
` + where + `
ORDER BY created_at DESC
LIMIT ` + limitPlaceholder
	rows, err := r.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Invitation, 0)
	for rows.Next() {
		item, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r Repository) RevokeInvitation(ctx context.Context, tenantID, invitationID string) (Invitation, error) {
	const q = `
UPDATE tenant_invitations
SET status = 'revoked',
    revoked_at = now(),
    updated_at = now()
WHERE id = $2 AND tenant_id = $1 AND status = 'pending' AND deleted_at IS NULL
RETURNING id::text, tenant_id::text, email::text, role_code, status, COALESCE(invited_by::text, ''),
          COALESCE(accepted_by::text, ''), accepted_at, revoked_at, expired_at, created_at, updated_at`
	item, err := scanInvitation(r.DB.QueryRow(ctx, q, tenantID, invitationID))
	if err == pgx.ErrNoRows {
		return Invitation{}, ErrInvitationNotFound
	}
	return item, err
}

func (r Repository) AcceptInvitation(ctx context.Context, userID, userEmail string, req AcceptInvitationRequest) (Tenant, error) {
	req.Normalize()
	if err := req.Validate(); err != nil {
		return Tenant{}, err
	}
	tokenHash := hashInvitationToken(req.Token)
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return Tenant{}, err
	}
	defer tx.Rollback(ctx)

	const invitationQ = `
SELECT id::text, tenant_id::text, email::text, role_code, status, COALESCE(invited_by::text, ''),
       COALESCE(accepted_by::text, ''), accepted_at, revoked_at, expired_at, created_at, updated_at
FROM tenant_invitations
WHERE token_hash = $1 AND deleted_at IS NULL
FOR UPDATE`
	inv, err := scanInvitation(tx.QueryRow(ctx, invitationQ, tokenHash))
	if err == pgx.ErrNoRows {
		return Tenant{}, ErrInvitationNotFound
	}
	if err != nil {
		return Tenant{}, err
	}
	if inv.Status != "pending" {
		return Tenant{}, ErrInvitationNotPending
	}
	if time.Now().UTC().After(inv.ExpiredAt) {
		const expireQ = `UPDATE tenant_invitations SET status = 'expired', updated_at = now() WHERE id = $1`
		_, _ = tx.Exec(ctx, expireQ, inv.ID)
		return Tenant{}, ErrInvitationExpired
	}
	if strings.ToLower(strings.TrimSpace(userEmail)) != strings.ToLower(strings.TrimSpace(inv.Email)) {
		return Tenant{}, ErrInvitationEmailMismatch
	}
	const upsertMember = `
INSERT INTO tenant_members(tenant_id, user_id, role_code, status)
VALUES ($1, $2, $3, 'active')
ON CONFLICT (tenant_id, user_id) DO UPDATE
SET role_code = EXCLUDED.role_code,
    status = 'active',
    deleted_at = NULL,
    updated_at = now()`
	if _, err := tx.Exec(ctx, upsertMember, inv.TenantID, userID, inv.RoleCode); err != nil {
		return Tenant{}, err
	}
	const acceptQ = `
UPDATE tenant_invitations
SET status = 'accepted',
    accepted_by = $2,
    accepted_at = now(),
    updated_at = now()
WHERE id = $1`
	if _, err := tx.Exec(ctx, acceptQ, inv.ID, userID); err != nil {
		return Tenant{}, err
	}
	const tenantQ = `
SELECT t.id::text, t.name, t.code, t.status, tm.role_code, t.created_at, t.updated_at
FROM tenants t
JOIN tenant_members tm ON tm.tenant_id = t.id
WHERE t.id = $1 AND tm.user_id = $2 AND tm.status = 'active' AND t.deleted_at IS NULL AND tm.deleted_at IS NULL`
	var t Tenant
	if err := tx.QueryRow(ctx, tenantQ, inv.TenantID, userID).Scan(&t.ID, &t.Name, &t.Code, &t.Status, &t.RoleCode, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return Tenant{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Tenant{}, err
	}
	return t, nil
}

func IsNotFound(err error) bool {
	return err == pgx.ErrNoRows
}

func jsonObject(v map[string]any) (string, error) {
	if v == nil {
		return "{}", nil
	}
	b, err := json.Marshal(v)
	return string(b), err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMember(row rowScanner) (Member, error) {
	var item Member
	err := row.Scan(
		&item.ID, &item.TenantID, &item.UserID, &item.Email, &item.Nickname,
		&item.RoleCode, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func scanInvitation(row rowScanner) (Invitation, error) {
	var item Invitation
	err := row.Scan(
		&item.ID, &item.TenantID, &item.Email, &item.RoleCode, &item.Status,
		&item.InvitedBy, &item.AcceptedBy, &item.AcceptedAt, &item.RevokedAt,
		&item.ExpiredAt, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

var ErrUserNotFound = errors.New("user not found")
var ErrMemberNotFound = errors.New("tenant member not found")
var ErrCannotRemoveOwner = errors.New("owner member cannot be removed")
var ErrInvitationNotFound = errors.New("tenant invitation not found")
var ErrInvitationNotPending = errors.New("tenant invitation is not pending")
var ErrInvitationExpired = errors.New("tenant invitation is expired")
var ErrInvitationEmailMismatch = errors.New("tenant invitation email mismatch")
