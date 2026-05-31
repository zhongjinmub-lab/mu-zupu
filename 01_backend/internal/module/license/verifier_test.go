package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

func TestVerifierVerifiesSignedLicense(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	expires := time.Date(2026, 12, 31, 15, 59, 59, 0, time.UTC)
	item := License{
		TenantID:    "tenant-1",
		LicenseNo:   "LIC-SIGNED",
		LicenseType: TypeTenant,
		Subject:     map[string]any{"tenant_name": "demo"},
		Limits:      map[string]any{MetricRAGRequestsForTest: float64(10)},
		PublicKeyID: "default",
		ExpiredAt:   &expires,
	}
	payload, err := LicensePayload(item)
	if err != nil {
		t.Fatal(err)
	}
	item.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload))

	verifier, err := NewVerifierFromConfig(`{"default":"` + base64.StdEncoding.EncodeToString(pub) + `"}`)
	if err != nil {
		t.Fatal(err)
	}
	result := verifier.Verify(item)
	if !result.Valid {
		t.Fatalf("expected valid signature: %#v", result)
	}
	if result.Mode != "offline" || !result.HasSignature || result.PublicKeyID != "default" {
		t.Fatalf("unexpected offline verify summary: %#v", result)
	}

	item.Limits["rag_requests"] = float64(11)
	result = verifier.Verify(item)
	if result.Valid {
		t.Fatal("expected tampered license to fail verification")
	}
}

func TestVerifierAcceptsOnlineLicenseWithoutSignature(t *testing.T) {
	verifier, err := NewVerifierFromConfig("")
	if err != nil {
		t.Fatal(err)
	}
	item := License{LicenseType: TypeTenant, Status: StatusInactive}
	result := verifier.Verify(item)
	if !result.Valid || result.Mode != "online" || result.HasSignature {
		t.Fatalf("expected online license to pass status verification: %#v", result)
	}
}

func TestVerifierRejectsExpiredOnlineLicense(t *testing.T) {
	verifier, err := NewVerifierFromConfig("")
	if err != nil {
		t.Fatal(err)
	}
	expired := time.Now().Add(-time.Minute)
	result := verifier.Verify(License{LicenseType: TypeTrial, ExpiredAt: &expired})
	if result.Valid || result.Mode != "online" || result.Message == "" {
		t.Fatalf("expected expired online license to fail: %#v", result)
	}
}

func TestVerifierRejectsExpiredStatus(t *testing.T) {
	verifier, err := NewVerifierFromConfig("")
	if err != nil {
		t.Fatal(err)
	}
	result := verifier.Verify(License{LicenseType: TypeTenant, Status: StatusExpired})
	if result.Valid || result.Message == "" {
		t.Fatalf("expected expired status to fail: %#v", result)
	}
}

func TestVerifierRejectsMissingKey(t *testing.T) {
	verifier, err := NewVerifierFromConfig("")
	if err != nil {
		t.Fatal(err)
	}
	result := verifier.Verify(License{LicenseType: TypeOffline, PublicKeyID: "missing", Signature: "bad"})
	if result.Valid || result.Message == "" {
		t.Fatalf("expected missing key failure: %#v", result)
	}
}

func TestNewVerifierRejectsInvalidPublicKey(t *testing.T) {
	if _, err := NewVerifierFromConfig(`{"bad":"abc"}`); err == nil {
		t.Fatal("expected invalid public key error")
	}
}

const MetricRAGRequestsForTest = "rag_requests"
