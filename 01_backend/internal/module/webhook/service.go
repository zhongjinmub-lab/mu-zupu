package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type Service struct {
	Repo   Repository
	Client *http.Client
}

func NewService(repo Repository) Service {
	return Service{
		Repo:   repo,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s Service) Emit(ctx context.Context, tenantID, eventType string, payload map[string]any) {
	if s.Repo.DB == nil || tenantID == "" || eventType == "" {
		return
	}
	endpoints, err := s.Repo.ListActiveEndpointsForEvent(ctx, tenantID, eventType)
	if err != nil {
		return
	}
	event := Event{Type: eventType, TenantID: tenantID, Payload: payload, CreatedAt: time.Now().UTC()}
	for _, endpoint := range endpoints {
		_, _ = s.Deliver(ctx, endpoint, event)
	}
}

func (s Service) Test(ctx context.Context, tenantID, endpointID string) (Delivery, error) {
	endpoint, err := s.Repo.GetEndpoint(ctx, tenantID, endpointID)
	if err != nil {
		return Delivery{}, err
	}
	event := Event{
		Type:      EventWebhookTest,
		TenantID:  tenantID,
		Payload:   map[string]any{"message": "Webhook 测试发送", "endpoint_id": endpointID},
		CreatedAt: time.Now().UTC(),
	}
	return s.Deliver(ctx, endpoint, event)
}

func (s Service) Deliver(ctx context.Context, endpoint Endpoint, event Event) (Delivery, error) {
	body, _ := json.Marshal(event)
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(body))
	if err != nil {
		return s.record(ctx, endpoint, event, body, 0, "", err.Error(), time.Since(start))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "mu-agent-saas-webhook/1.0")
	req.Header.Set("X-Webhook-Event", event.Type)
	if endpoint.Secret != "" {
		req.Header.Set("X-Webhook-Signature", signBody(endpoint.Secret, body))
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return s.record(ctx, endpoint, event, body, 0, "", err.Error(), time.Since(start))
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	errMsg := ""
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg = resp.Status
	}
	return s.record(ctx, endpoint, event, body, resp.StatusCode, string(respBody), errMsg, time.Since(start))
}

func (s Service) record(ctx context.Context, endpoint Endpoint, event Event, body []byte, httpStatus int, responseBody, errMessage string, cost time.Duration) (Delivery, error) {
	var requestBody map[string]any
	_ = json.Unmarshal(body, &requestBody)
	status := "success"
	if errMessage != "" || httpStatus == 0 || httpStatus >= 300 {
		status = "failed"
	}
	return s.Repo.CreateDelivery(ctx, Delivery{
		TenantID:     event.TenantID,
		EndpointID:   endpoint.ID,
		EventType:    event.Type,
		TargetURL:    endpoint.URL,
		Status:       status,
		HTTPStatus:   httpStatus,
		RequestBody:  requestBody,
		ResponseBody: responseBody,
		ErrorMessage: errMessage,
		DurationMS:   cost.Milliseconds(),
		RetryCount:   0,
	})
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
