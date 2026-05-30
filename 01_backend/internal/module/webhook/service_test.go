package webhook

import (
	"context"
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
