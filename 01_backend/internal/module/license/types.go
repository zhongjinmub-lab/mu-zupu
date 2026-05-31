package license

import (
	"errors"
	"strings"
	"time"
)

const (
	StatusInactive = "inactive"
	StatusActive   = "active"
	StatusRevoked  = "revoked"
	StatusExpired  = "expired"

	TypeTenant  = "tenant"
	TypeTrial   = "trial"
	TypeOffline = "offline"
)

type License struct {
	ID           string         `json:"id"`
	TenantID     string         `json:"tenant_id"`
	LicenseNo    string         `json:"license_no"`
	LicenseType  string         `json:"license_type"`
	Status       string         `json:"status"`
	Subject      map[string]any `json:"subject"`
	Limits       map[string]any `json:"limits"`
	PublicKeyID  string         `json:"public_key_id,omitempty"`
	HasSignature bool           `json:"has_signature"`
	Signature    string         `json:"-"`
	IssuedAt     time.Time      `json:"issued_at"`
	ActivatedAt  *time.Time     `json:"activated_at,omitempty"`
	RevokedAt    *time.Time     `json:"revoked_at,omitempty"`
	ExpiredAt    *time.Time     `json:"expired_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type CreateLicenseRequest struct {
	LicenseNo   string         `json:"license_no"`
	LicenseType string         `json:"license_type"`
	Subject     map[string]any `json:"subject"`
	Limits      map[string]any `json:"limits"`
	PublicKeyID string         `json:"public_key_id"`
	Signature   string         `json:"signature"`
	ExpiredAt   *time.Time     `json:"expired_at"`
}

func (r *CreateLicenseRequest) Normalize() {
	r.LicenseNo = strings.ToUpper(strings.TrimSpace(r.LicenseNo))
	r.LicenseType = strings.ToLower(strings.TrimSpace(r.LicenseType))
	r.PublicKeyID = strings.TrimSpace(r.PublicKeyID)
	r.Signature = strings.TrimSpace(r.Signature)
	if r.LicenseType == "" {
		r.LicenseType = TypeTenant
	}
	if r.Subject == nil {
		r.Subject = map[string]any{}
	}
	if r.Limits == nil {
		r.Limits = map[string]any{}
	}
}

func (r CreateLicenseRequest) Validate() error {
	if r.LicenseNo != "" && len(r.LicenseNo) > 128 {
		return errors.New("license_no must be at most 128 characters")
	}
	switch r.LicenseType {
	case TypeTenant, TypeTrial, TypeOffline:
	default:
		return errors.New("license_type must be tenant, trial or offline")
	}
	if r.ExpiredAt != nil && !r.ExpiredAt.After(time.Now()) {
		return errors.New("expired_at must be in the future")
	}
	return nil
}
