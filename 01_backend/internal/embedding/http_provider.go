package embedding

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
		model = "text-embedding-3-small"
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("embedding base url is required")
	}
	timeout := cfg.TimeoutSeconds
	if timeout <= 0 {
		timeout = 30
	}
	return HTTPProvider{
		name:   name,
		model:  model,
		url:    baseURL + "/embeddings",
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

func (p HTTPProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(map[string]any{
		"model": p.model,
		"input": text,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding provider status %d", resp.StatusCode)
	}
	var out struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if len(out.Data) == 0 {
		return nil, errors.New("embedding provider returned empty data")
	}
	if len(out.Data[0].Embedding) != Dim {
		return nil, fmt.Errorf("embedding dimension must be %d", Dim)
	}
	return out.Data[0].Embedding, nil
}
