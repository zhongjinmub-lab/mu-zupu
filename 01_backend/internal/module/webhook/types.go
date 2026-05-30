package webhook

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"

	EventWebhookTest       = "webhook.test"
	EventOrderPaid         = "order.paid"
	EventLicenseActivated  = "license.activated"
	EventLicenseRevoked    = "license.revoked"
	EventAgentChatFinished = "agent.chat.finished"
)

type Endpoint struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Secret    string    `json:"secret,omitempty"`
	Events    []string  `json:"events"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Delivery struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenant_id"`
	EndpointID    string         `json:"endpoint_id,omitempty"`
	EventType     string         `json:"event_type"`
	TargetURL     string         `json:"target_url"`
	Status        string         `json:"status"`
	HTTPStatus    int            `json:"http_status,omitempty"`
	RequestBody   map[string]any `json:"request_body"`
	ResponseBody  string         `json:"response_body,omitempty"`
	ErrorMessage  string         `json:"error_message,omitempty"`
	DurationMS    int64          `json:"duration_ms"`
	RetryCount    int            `json:"retry_count"`
	NextRetryAt   *time.Time     `json:"next_retry_at,omitempty"`
	LastAttemptAt *time.Time     `json:"last_attempt_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
}

type CreateEndpointRequest struct {
	Name   string   `json:"name" binding:"required"`
	URL    string   `json:"url" binding:"required"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
	Status string   `json:"status"`
}

type UpdateEndpointRequest struct {
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	Secret string   `json:"secret"`
	Events []string `json:"events"`
	Status string   `json:"status"`
}

type Event struct {
	Type      string         `json:"type"`
	TenantID  string         `json:"tenant_id"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

type RetryJob struct {
	Delivery Delivery
	Endpoint Endpoint
	Event    Event
}

type RetrySummary struct {
	Claimed   int `json:"claimed"`
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
}

func (r *CreateEndpointRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.URL = strings.TrimSpace(r.URL)
	r.Secret = strings.TrimSpace(r.Secret)
	r.Status = normalizeStatus(r.Status)
	r.Events = normalizeEvents(r.Events)
	if len(r.Events) == 0 {
		r.Events = []string{EventWebhookTest}
	}
}

func (r CreateEndpointRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if err := validateURL(r.URL); err != nil {
		return err
	}
	if r.Status != StatusActive && r.Status != StatusDisabled {
		return errors.New("status must be active or disabled")
	}
	if len(r.Events) == 0 {
		return errors.New("events is required")
	}
	return nil
}

func (r *UpdateEndpointRequest) Normalize() {
	r.Name = strings.TrimSpace(r.Name)
	r.URL = strings.TrimSpace(r.URL)
	r.Secret = strings.TrimSpace(r.Secret)
	r.Status = normalizeStatus(r.Status)
	r.Events = normalizeEvents(r.Events)
}

func (r UpdateEndpointRequest) Validate() error {
	if r.URL != "" {
		if err := validateURL(r.URL); err != nil {
			return err
		}
	}
	if r.Status != "" && r.Status != StatusActive && r.Status != StatusDisabled {
		return errors.New("status must be active or disabled")
	}
	return nil
}

func normalizeStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return StatusActive
	}
	return status
}

func normalizeEvents(events []string) []string {
	seen := make(map[string]struct{}, len(events))
	out := make([]string, 0, len(events))
	for _, item := range events {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func validateURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errors.New("url must be a valid http or https address")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("url only supports http or https")
	}
	return nil
}

var ErrEndpointNotFound = errors.New("webhook endpoint not found")
