package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPConfig struct {
	Provider       string
	Model          string
	BaseURL        string
	APIKey         string
	TimeoutSeconds int
}

type HTTPProvider struct {
	name   string
	model  string
	url    string
	apiKey string
	client *http.Client
}

func NewHTTPProvider(cfg HTTPConfig) (Provider, error) {
	name := strings.TrimSpace(cfg.Provider)
	if name == "" {
		name = "openai_compatible"
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("generation base url is required")
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 60
	}
	return HTTPProvider{
		name:   name,
		model:  model,
		url:    baseURL + "/chat/completions",
		apiKey: strings.TrimSpace(cfg.APIKey),
		client: &http.Client{
			Timeout:   time.Duration(timeout) * time.Second,
			Transport: &http.Transport{DisableKeepAlives: true},
		},
	}, nil
}

func (p HTTPProvider) Name() string {
	return p.name
}

func (p HTTPProvider) Model() string {
	return p.model
}

func (p HTTPProvider) Generate(ctx context.Context, req Request) (Response, error) {
	body, err := json.Marshal(map[string]any{
		"model":       p.model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
	})
	if err != nil {
		return Response{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return Response{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("generation provider status %d", resp.StatusCode)
	}
	var out struct {
		Model   string `json:"model"`
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return Response{}, err
	}
	if len(out.Choices) == 0 || strings.TrimSpace(out.Choices[0].Message.Content) == "" {
		return Response{}, errors.New("generation provider returned empty answer")
	}
	model := out.Model
	if model == "" {
		model = p.model
	}
	return Response{Answer: strings.TrimSpace(out.Choices[0].Message.Content), Model: model}, nil
}
