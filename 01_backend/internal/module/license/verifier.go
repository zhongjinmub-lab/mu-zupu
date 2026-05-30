package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type Verifier struct {
	keys map[string]ed25519.PublicKey
}

type VerifyResult struct {
	Valid       bool   `json:"valid"`
	PublicKeyID string `json:"public_key_id,omitempty"`
	Message     string `json:"message,omitempty"`
}

type signedPayload struct {
	TenantID    string         `json:"tenant_id"`
	LicenseNo   string         `json:"license_no"`
	LicenseType string         `json:"license_type"`
	Subject     map[string]any `json:"subject"`
	Limits      map[string]any `json:"limits"`
	ExpiredAt   string         `json:"expired_at,omitempty"`
}

func NewVerifierFromConfig(raw string) (Verifier, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Verifier{keys: map[string]ed25519.PublicKey{}}, nil
	}
	var values map[string]string
	if strings.HasPrefix(raw, "{") {
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return Verifier{}, err
		}
	} else {
		values = map[string]string{}
		for _, part := range strings.Split(raw, ",") {
			k, v, ok := strings.Cut(part, "=")
			if !ok {
				return Verifier{}, fmt.Errorf("invalid LICENSE_PUBLIC_KEYS entry %q", part)
			}
			values[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	keys := make(map[string]ed25519.PublicKey, len(values))
	for id, encoded := range values {
		id = strings.TrimSpace(id)
		if id == "" {
			return Verifier{}, errors.New("license public key id is required")
		}
		key, err := decodeBase64(encoded)
		if err != nil {
			return Verifier{}, fmt.Errorf("invalid license public key %q: %w", id, err)
		}
		if len(key) != ed25519.PublicKeySize {
			return Verifier{}, fmt.Errorf("invalid license public key %q size", id)
		}
		keys[id] = ed25519.PublicKey(key)
	}
	return Verifier{keys: keys}, nil
}

func (v Verifier) Verify(item License) VerifyResult {
	if item.Signature == "" {
		return VerifyResult{Valid: false, PublicKeyID: item.PublicKeyID, Message: "signature is required"}
	}
	if item.PublicKeyID == "" {
		return VerifyResult{Valid: false, Message: "public_key_id is required"}
	}
	key, ok := v.keys[item.PublicKeyID]
	if !ok {
		return VerifyResult{Valid: false, PublicKeyID: item.PublicKeyID, Message: "public key is not configured"}
	}
	sig, err := decodeBase64(item.Signature)
	if err != nil {
		return VerifyResult{Valid: false, PublicKeyID: item.PublicKeyID, Message: "invalid signature encoding"}
	}
	payload, err := LicensePayload(item)
	if err != nil {
		return VerifyResult{Valid: false, PublicKeyID: item.PublicKeyID, Message: err.Error()}
	}
	if !ed25519.Verify(key, payload, sig) {
		return VerifyResult{Valid: false, PublicKeyID: item.PublicKeyID, Message: "signature verification failed"}
	}
	if item.ExpiredAt != nil && !item.ExpiredAt.After(time.Now()) {
		return VerifyResult{Valid: false, PublicKeyID: item.PublicKeyID, Message: "license is expired"}
	}
	if item.Status == StatusRevoked {
		return VerifyResult{Valid: false, PublicKeyID: item.PublicKeyID, Message: "license is revoked"}
	}
	return VerifyResult{Valid: true, PublicKeyID: item.PublicKeyID, Message: "ok"}
}

func LicensePayload(item License) ([]byte, error) {
	payload := signedPayload{
		TenantID:    item.TenantID,
		LicenseNo:   item.LicenseNo,
		LicenseType: item.LicenseType,
		Subject:     sortedMap(item.Subject),
		Limits:      sortedMap(item.Limits),
	}
	if item.ExpiredAt != nil {
		payload.ExpiredAt = item.ExpiredAt.UTC().Format(time.RFC3339)
	}
	return json.Marshal(payload)
}

func decodeBase64(v string) ([]byte, error) {
	v = strings.TrimSpace(v)
	if out, err := base64.StdEncoding.DecodeString(v); err == nil {
		return out, nil
	}
	return base64.RawStdEncoding.DecodeString(v)
}

func sortedMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(map[string]any, len(in))
	for _, k := range keys {
		out[k] = in[k]
	}
	return out
}
