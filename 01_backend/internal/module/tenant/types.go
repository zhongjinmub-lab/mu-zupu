package tenant

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const ContextTenantKey = "current_tenant"

type Tenant struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	Status    string    `json:"status"`
	RoleCode  string    `json:"role_code,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name string `json:"name" binding:"required"`
	Code string `json:"code" binding:"required"`
}

type Member struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Nickname  string    `json:"nickname"`
	RoleCode  string    `json:"role_code"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AddMemberRequest struct {
	Email    string `json:"email" binding:"required,email"`
	RoleCode string `json:"role_code"`
}

type UpdateMemberRoleRequest struct {
	RoleCode string `json:"role_code" binding:"required"`
}

type AuditLog struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id,omitempty"`
	ActorUserID  string         `json:"actor_user_id,omitempty"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type,omitempty"`
	ResourceID   string         `json:"resource_id,omitempty"`
	IP           string         `json:"ip,omitempty"`
	UserAgent    string         `json:"user_agent,omitempty"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
}

type AuditLogQuery struct {
	TenantID     string
	Action       string
	ResourceType string
	ActorUserID  string
	From         time.Time
	To           time.Time
	Cursor       AuditLogCursor
	CursorRaw    string
	Limit        int
}

type AuditLogCursor struct {
	CreatedAt time.Time
	ID        string
}

type Invitation struct {
	ID         string     `json:"id"`
	TenantID   string     `json:"tenant_id"`
	Email      string     `json:"email"`
	RoleCode   string     `json:"role_code"`
	Status     string     `json:"status"`
	InvitedBy  string     `json:"invited_by,omitempty"`
	AcceptedBy string     `json:"accepted_by,omitempty"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	ExpiredAt  time.Time  `json:"expired_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Token      string     `json:"token,omitempty"`
}

type CreateInvitationRequest struct {
	Email    string `json:"email" binding:"required,email"`
	RoleCode string `json:"role_code"`
	TTLHours int    `json:"ttl_hours"`
}

type AcceptInvitationRequest struct {
	Token string `json:"token" binding:"required"`
}

func (r *AddMemberRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.RoleCode = normalizeRoleCode(r.RoleCode)
}

func (r AddMemberRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	if !validRoleCode(r.RoleCode) {
		return errors.New("role_code must be owner, admin, member or viewer")
	}
	return nil
}

func (r *UpdateMemberRoleRequest) Normalize() {
	r.RoleCode = normalizeRoleCode(r.RoleCode)
}

func (r UpdateMemberRoleRequest) Validate() error {
	if !validRoleCode(r.RoleCode) {
		return errors.New("role_code must be owner, admin, member or viewer")
	}
	return nil
}

func (r *CreateInvitationRequest) Normalize() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.RoleCode = normalizeInviteRoleCode(r.RoleCode)
	if r.TTLHours == 0 {
		r.TTLHours = 24 * 7
	}
}

func (r CreateInvitationRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	if !validInviteRoleCode(r.RoleCode) {
		return errors.New("role_code must be admin, member or viewer")
	}
	if r.TTLHours < 1 || r.TTLHours > 24*30 {
		return errors.New("ttl_hours must be between 1 and 720")
	}
	return nil
}

func (r *AcceptInvitationRequest) Normalize() {
	r.Token = strings.TrimSpace(r.Token)
}

func (r AcceptInvitationRequest) Validate() error {
	if r.Token == "" {
		return errors.New("token is required")
	}
	return nil
}

func (q *AuditLogQuery) Normalize() {
	q.TenantID = strings.TrimSpace(q.TenantID)
	q.Action = strings.TrimSpace(q.Action)
	q.ResourceType = strings.TrimSpace(q.ResourceType)
	q.ActorUserID = strings.TrimSpace(q.ActorUserID)
	q.CursorRaw = strings.TrimSpace(q.CursorRaw)
	if q.Limit <= 0 || q.Limit > 100 {
		q.Limit = 50
	}
}

func (q *AuditLogQuery) Validate() error {
	if q.TenantID == "" {
		return errors.New("tenant_id is required")
	}
	if q.ActorUserID != "" {
		if _, err := uuid.Parse(q.ActorUserID); err != nil {
			return errors.New("actor_user_id must be a valid uuid")
		}
	}
	if !q.From.IsZero() && !q.To.IsZero() && q.From.After(q.To) {
		return errors.New("from must be before or equal to to")
	}
	if q.CursorRaw != "" {
		cursor, err := DecodeAuditLogCursor(q.CursorRaw)
		if err != nil {
			return err
		}
		q.Cursor = cursor
	}
	return nil
}

func ParseAuditLogTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", v)
}

func EncodeAuditLogCursor(item AuditLog) string {
	if item.ID == "" || item.CreatedAt.IsZero() {
		return ""
	}
	payload := item.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + item.ID
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func DecodeAuditLogCursor(raw string) (AuditLogCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return AuditLogCursor{}, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return AuditLogCursor{}, errors.New("cursor is invalid")
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return AuditLogCursor{}, errors.New("cursor is invalid")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return AuditLogCursor{}, errors.New("cursor is invalid")
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		return AuditLogCursor{}, errors.New("cursor is invalid")
	}
	return AuditLogCursor{CreatedAt: createdAt, ID: parts[1]}, nil
}

func normalizeCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, " ", "-")
	return code
}

func normalizeRoleCode(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return "member"
	}
	return role
}

func validRoleCode(role string) bool {
	switch role {
	case "owner", "admin", "member", "viewer":
		return true
	default:
		return false
	}
}

func normalizeInviteRoleCode(role string) string {
	role = normalizeRoleCode(role)
	if role == "owner" {
		return "admin"
	}
	return role
}

func validInviteRoleCode(role string) bool {
	switch role {
	case "admin", "member", "viewer":
		return true
	default:
		return false
	}
}

func CanManageMembers(role string) bool {
	return role == "owner" || role == "admin"
}

func newInvitationToken() (string, string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	token := hex.EncodeToString(b[:])
	return token, hashInvitationToken(token), nil
}

func hashInvitationToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}
