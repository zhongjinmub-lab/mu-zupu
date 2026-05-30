package license

import (
	"strings"
	"testing"
	"time"
)

func TestCreateLicenseRequestNormalizeAndValidate(t *testing.T) {
	req := CreateLicenseRequest{
		LicenseNo:   " lic-test ",
		LicenseType: "",
		PublicKeyID: " key-1 ",
		Signature:   " sig ",
	}
	req.Normalize()
	if req.LicenseNo != "LIC-TEST" || req.LicenseType != TypeTenant || req.PublicKeyID != "key-1" || req.Signature != "sig" {
		t.Fatalf("normalized request = %#v", req)
	}
	if req.Subject == nil || req.Limits == nil {
		t.Fatalf("expected default json objects: %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid license request: %v", err)
	}
}

func TestCreateLicenseRequestValidateRejectsInvalidType(t *testing.T) {
	req := CreateLicenseRequest{LicenseType: "bad"}
	req.Normalize()
	if err := req.Validate(); err == nil {
		t.Fatal("expected invalid license_type error")
	}
}

func TestCreateLicenseRequestValidateRejectsPastExpiry(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	req := CreateLicenseRequest{ExpiredAt: &past}
	req.Normalize()
	if err := req.Validate(); err == nil {
		t.Fatal("expected expired_at validation error")
	}
}

func TestNewLicenseNo(t *testing.T) {
	no := newLicenseNo()
	if !strings.HasPrefix(no, "LIC-") || len(no) != 28 {
		t.Fatalf("license_no = %q", no)
	}
}
