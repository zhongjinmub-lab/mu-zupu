package generation

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestLocalProviderGenerate(t *testing.T) {
	p, err := NewProvider("local", "")
	if err != nil {
		t.Fatalf("NewProvider error: %v", err)
	}
	out, err := p.Generate(context.Background(), Request{
		Messages: []Message{
			{Role: "system", Content: "上下文 A"},
			{Role: "user", Content: "问题 B"},
		},
	})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if out.Answer == "" || out.Model != "local-rag" {
		t.Fatalf("response = %#v", out)
	}
}

func TestHTTPProviderGenerate(t *testing.T) {
	p := HTTPProvider{
		name:   "openai_compatible",
		model:  "test-chat",
		url:    "https://llm.example/v1/chat/completions",
		apiKey: "test-key",
		client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://llm.example/v1/chat/completions" {
				t.Fatalf("url = %q", r.URL.String())
			}
			if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
				t.Fatalf("authorization = %q", got)
			}
			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			if req["model"] != "test-chat" {
				t.Fatalf("request = %#v", req)
			}
			body, _ := json.Marshal(map[string]any{
				"model": "test-chat",
				"choices": []map[string]any{{
					"message": map[string]any{"role": "assistant", "content": "answer"},
				}},
			})
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(body)),
				Header:     make(http.Header),
			}, nil
		})},
	}
	out, err := p.Generate(context.Background(), Request{Messages: []Message{{Role: "user", Content: "hello"}}})
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	if out.Answer != "answer" || out.Model != "test-chat" {
		t.Fatalf("response = %#v", out)
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
