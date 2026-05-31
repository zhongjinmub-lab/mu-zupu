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
	Repo             Repository
	Client           *http.Client
	MaxRetries       int
	RetryBaseSeconds int
}

type ServiceOptions struct {
	MaxRetries       int
	RetryBaseSeconds int
}

func NewService(repo Repository, opts ...ServiceOptions) Service {
	options := ServiceOptions{MaxRetries: 3, RetryBaseSeconds: 60}
	if len(opts) > 0 {
		options = opts[0]
	}
	if options.MaxRetries < 0 {
		options.MaxRetries = 0
	}
	if options.RetryBaseSeconds <= 0 {
		options.RetryBaseSeconds = 60
	}
	return Service{
		Repo:             repo,
		Client:           &http.Client{Timeout: 5 * time.Second},
		MaxRetries:       options.MaxRetries,
		RetryBaseSeconds: options.RetryBaseSeconds,
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
	result := s.send(ctx, endpoint, event, 0)
	result.NextRetryAt = s.nextRetryAt(result)
	return s.Repo.CreateDelivery(ctx, result)
}

func (s Service) RetryDue(ctx context.Context, limit int) RetrySummary {
	var summary RetrySummary
	jobs, err := s.Repo.ClaimRetryJobs(ctx, limit)
	if err != nil {
		summary.Failed++
		return summary
	}
	summary.Claimed = len(jobs)
	for _, job := range jobs {
		result := s.send(ctx, job.Endpoint, job.Event, job.Delivery.RetryCount+1)
		result.NextRetryAt = s.nextRetryAt(result)
		if _, err := s.Repo.UpdateDeliveryAttempt(ctx, job.Delivery.ID, result); err != nil {
			summary.Failed++
			continue
		}
		if result.Status == "success" {
			summary.Processed++
		} else {
			summary.Failed++
		}
	}
	return summary
}

func (s Service) RetryDelivery(ctx context.Context, tenantID, deliveryID string) (Delivery, error) {
	job, err := s.Repo.GetRetryJob(ctx, tenantID, deliveryID)
	if err != nil {
		return Delivery{}, err
	}
	if job.Delivery.Status != "failed" {
		return Delivery{}, ErrDeliveryNotRetryable
	}
	if job.Endpoint.Status != StatusActive {
		return Delivery{}, ErrDeliveryNotRetryable
	}
	result := s.send(ctx, job.Endpoint, job.Event, job.Delivery.RetryCount+1)
	result.NextRetryAt = s.nextRetryAt(result)
	return s.Repo.UpdateDeliveryAttempt(ctx, job.Delivery.ID, result)
}

func (s Service) send(ctx context.Context, endpoint Endpoint, event Event, retryCount int) Delivery {
	body, _ := json.Marshal(event)
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.URL, bytes.NewReader(body))
	if err != nil {
		return s.deliveryResult(endpoint, event, body, 0, "", err.Error(), time.Since(start), retryCount)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "mu-agent-saas-webhook/1.0")
	req.Header.Set("X-Webhook-Event", event.Type)
	if endpoint.Secret != "" {
		req.Header.Set("X-Webhook-Signature", signBody(endpoint.Secret, body))
	}
	resp, err := s.Client.Do(req)
	if err != nil {
		return s.deliveryResult(endpoint, event, body, 0, "", err.Error(), time.Since(start), retryCount)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	errMsg := ""
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg = resp.Status
	}
	return s.deliveryResult(endpoint, event, body, resp.StatusCode, string(respBody), errMsg, time.Since(start), retryCount)
}

func (s Service) deliveryResult(endpoint Endpoint, event Event, body []byte, httpStatus int, responseBody, errMessage string, cost time.Duration, retryCount int) Delivery {
	var requestBody map[string]any
	_ = json.Unmarshal(body, &requestBody)
	status := "success"
	if errMessage != "" || httpStatus == 0 || httpStatus >= 300 {
		status = "failed"
	}
	return Delivery{
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
		RetryCount:   retryCount,
	}
}

func (s Service) nextRetryAt(result Delivery) *time.Time {
	if result.Status != "failed" || result.RetryCount >= s.MaxRetries {
		return nil
	}
	delay := time.Duration(s.RetryBaseSeconds) * time.Second
	for i := 0; i < result.RetryCount; i++ {
		delay *= 2
	}
	next := time.Now().UTC().Add(delay)
	return &next
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
