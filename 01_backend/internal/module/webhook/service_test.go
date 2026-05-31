package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSendAddsSignatureAndMarksSuccess(t *testing.T) {
	var gotSignature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSignature = r.Header.Get("X-Webhook-Signature")
		if r.Header.Get("X-Webhook-Event") != EventWebhookTest {
			t.Fatalf("unexpected event header: %s", r.Header.Get("X-Webhook-Event"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	service := NewService(Repository{})
	result := service.send(context.Background(), Endpoint{ID: "endpoint-1", URL: server.URL, Secret: "secret"}, Event{
		Type:      EventWebhookTest,
		TenantID:  "tenant-1",
		Payload:   map[string]any{"message": "test"},
		CreatedAt: time.Now().UTC(),
	}, 0)

	if result.Status != "success" {
		t.Fatalf("expected success, got %s", result.Status)
	}
	if result.HTTPStatus != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", result.HTTPStatus)
	}
	if !strings.HasPrefix(gotSignature, "sha256=") {
		t.Fatalf("expected sha256 signature, got %q", gotSignature)
	}
	if result.RetryCount != 0 {
		t.Fatalf("expected retry_count 0, got %d", result.RetryCount)
	}
}

func TestFailedDeliverySchedulesRetry(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary failure", http.StatusBadGateway)
	}))
	defer server.Close()

	service := NewService(Repository{}, ServiceOptions{MaxRetries: 3, RetryBaseSeconds: 1})
	result := service.send(context.Background(), Endpoint{ID: "endpoint-1", URL: server.URL}, Event{
		Type:      EventOrderPaid,
		TenantID:  "tenant-1",
		Payload:   map[string]any{"order_id": "order-1"},
		CreatedAt: time.Now().UTC(),
	}, 1)
	result.NextRetryAt = service.nextRetryAt(result)

	if result.Status != "failed" {
		t.Fatalf("expected failed, got %s", result.Status)
	}
	if result.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", result.HTTPStatus)
	}
	if result.NextRetryAt == nil {
		t.Fatal("expected next_retry_at")
	}
	if !result.NextRetryAt.After(time.Now().UTC()) {
		t.Fatalf("expected next_retry_at in future, got %s", result.NextRetryAt)
	}
}

func TestMaxRetryStopsScheduling(t *testing.T) {
	service := NewService(Repository{}, ServiceOptions{MaxRetries: 2, RetryBaseSeconds: 1})
	next := service.nextRetryAt(Delivery{Status: "failed", RetryCount: 2})
	if next != nil {
		t.Fatalf("expected no next retry, got %s", next)
	}
	next = service.nextRetryAt(Delivery{Status: "success", RetryCount: 0})
	if next != nil {
		t.Fatalf("expected no retry for success, got %s", next)
	}
}

func TestEndpointJSONHidesSecret(t *testing.T) {
	body, err := json.Marshal(Endpoint{
		ID:        "endpoint-1",
		TenantID:  "tenant-1",
		Name:      "支付通知",
		URL:       "https://example.com/webhook",
		Secret:    "should-not-leak",
		HasSecret: true,
		Events:    []string{EventOrderPaid},
		Status:    StatusActive,
	})
	if err != nil {
		t.Fatalf("marshal endpoint: %v", err)
	}
	text := string(body)
	if strings.Contains(text, "should-not-leak") || strings.Contains(text, "\"secret\"") {
		t.Fatalf("expected endpoint json hide secret, got %s", text)
	}
	if !strings.Contains(text, "\"has_secret\":true") {
		t.Fatalf("expected endpoint json include has_secret, got %s", text)
	}
}

func TestDeliveryQueryNormalizeAndValidate(t *testing.T) {
	query := DeliveryQuery{
		TenantID:   " tenant-1 ",
		EndpointID: " 11111111-1111-1111-1111-111111111111 ",
		EventType:  " ORDER.PAID ",
		Status:     " FAILED ",
		Limit:      500,
	}
	query.Normalize()

	if query.TenantID != "tenant-1" || query.EndpointID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected normalized ids: %#v", query)
	}
	if query.EventType != EventOrderPaid || query.Status != "failed" {
		t.Fatalf("unexpected normalized filters: %#v", query)
	}
	if query.Limit != 50 {
		t.Fatalf("expected default limit 50, got %d", query.Limit)
	}
	if err := query.Validate(); err != nil {
		t.Fatalf("expected valid query, got %v", err)
	}

	query.Status = "unknown"
	if err := query.Validate(); err == nil {
		t.Fatal("expected invalid status error")
	}

	query.Status = "failed"
	query.EndpointID = "bad-endpoint-id"
	if err := query.Validate(); err == nil {
		t.Fatal("expected invalid endpoint_id error")
	}
}

func TestDeliveryQueryNormalizeForExportAllowsLargerLimit(t *testing.T) {
	query := DeliveryQuery{
		TenantID:  " tenant-1 ",
		EventType: " WEBHOOK.TEST ",
		Status:    " SUCCESS ",
		Limit:     800,
	}
	query.NormalizeForExport()

	if query.TenantID != "tenant-1" || query.EventType != EventWebhookTest || query.Status != "success" {
		t.Fatalf("unexpected normalized export query: %#v", query)
	}
	if query.Limit != 800 {
		t.Fatalf("expected export limit 800, got %d", query.Limit)
	}

	query.Limit = 2000
	query.NormalizeForExport()
	if query.Limit != 1000 {
		t.Fatalf("expected export limit capped at 1000, got %d", query.Limit)
	}
}

func TestDeliveryQueryValidateTimeRange(t *testing.T) {
	query := DeliveryQuery{
		TenantID: "tenant-1",
		From:     time.Date(2026, 5, 31, 10, 0, 0, 0, time.UTC),
		To:       time.Date(2026, 5, 31, 9, 0, 0, 0, time.UTC),
	}
	if err := query.Validate(); err == nil {
		t.Fatal("expected invalid time range")
	}
	query.To = query.From.Add(time.Hour)
	if err := query.Validate(); err != nil {
		t.Fatalf("expected valid time range, got %v", err)
	}
}

func TestParseDeliveryTimeSupportsDateAndRFC3339(t *testing.T) {
	day, err := parseDeliveryTime("2026-05-31")
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	if day.Year() != 2026 || day.Month() != 5 || day.Day() != 31 {
		t.Fatalf("unexpected date: %s", day)
	}
	instant, err := parseDeliveryTime("2026-05-31T10:20:30Z")
	if err != nil {
		t.Fatalf("parse rfc3339: %v", err)
	}
	if instant.Hour() != 10 || instant.Minute() != 20 {
		t.Fatalf("unexpected instant: %s", instant)
	}
	if _, err := parseDeliveryTime("bad-time"); err == nil {
		t.Fatal("expected invalid time")
	}
}
