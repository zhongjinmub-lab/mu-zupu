package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestLocalProviderEmbedding(t *testing.T) {
	p, err := NewProvider("local", "")
	if err != nil {
		t.Fatalf("NewProvider error: %v", err)
	}
	a, err := p.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed error: %v", err)
	}
	b, _ := p.Embed(context.Background(), "hello")
	if len(a) != Dim {
		t.Fatalf("dim = %d", len(a))
	}
	if a[0] != b[0] || a[100] != b[100] {
		t.Fatal("local embedding must be deterministic")
	}
	if p.Name() != "local" {
		t.Fatalf("provider name = %q", p.Name())
	}
}

func TestHTTPProviderEmbedding(t *testing.T) {
	embedding := make([]float32, Dim)
	embedding[0] = 0.5
	p := HTTPProvider{
		name:   "openai_compatible",
		model:  "test-model",
		url:    "https://embedding.example/v1/embeddings",
		apiKey: "test-key",
		client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://embedding.example/v1/embeddings" {
				t.Fatalf("url = %q", r.URL.String())
			}
			if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
				t.Fatalf("authorization = %q", got)
			}
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if req["model"] != "test-model" || req["input"] != "hello" {
				t.Fatalf("request = %#v", req)
			}
			body, _ := json.Marshal(map[string]any{
				"data": []map[string]any{{"embedding": embedding}},
			})
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		})},
	}

	vec, err := p.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed error: %v", err)
	}
	if len(vec) != Dim || vec[0] != 0.5 {
		t.Fatalf("embedding result len=%d first=%v", len(vec), vec[0])
	}
	if p.Name() != "openai_compatible" || p.Model() != "test-model" {
		t.Fatalf("provider metadata name=%q model=%q", p.Name(), p.Model())
	}
}

func TestHTTPProviderRejectsWrongDimension(t *testing.T) {
	p := HTTPProvider{
		name:  "openai_compatible",
		model: "test-model",
		url:   "https://embedding.example/v1/embeddings",
		client: &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			body, _ := json.Marshal(map[string]any{
				"data": []map[string]any{{"embedding": []float32{1, 2, 3}}},
			})
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		})},
	}

	if _, err := p.Embed(context.Background(), "hello"); err == nil {
		t.Fatal("expected dimension error")
	}
}

func TestNewHTTPProviderRequiresBaseURL(t *testing.T) {
	if _, err := NewHTTPProvider(HTTPConfig{}); err == nil {
		t.Fatal("expected base url error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
