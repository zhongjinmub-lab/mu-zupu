package auth

import "testing"

func TestJWTSignAndParse(t *testing.T) {
	svc, err := NewJWTService("12345678901234567890123456789012", "test", 1)
	if err != nil {
		t.Fatalf("NewJWTService error: %v", err)
	}
	token, _, err := svc.Sign(User{ID: "user-1", Email: "a@example.com"})
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}
	claims, err := svc.Parse(token)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if claims.UserID != "user-1" || claims.Email != "a@example.com" {
		t.Fatalf("claims = %#v", claims)
	}
}

func TestJWTRequiresNonHardcodedSecret(t *testing.T) {
	if _, err := NewJWTService("", "test", 1); err == nil {
		t.Fatal("expected empty secret to fail")
	}
	if _, err := NewJWTService("short", "test", 1); err == nil {
		t.Fatal("expected short secret to fail")
	}
}
