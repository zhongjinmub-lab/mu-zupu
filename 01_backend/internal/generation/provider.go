package generation

import (
	"context"
	"errors"
	"strings"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Messages    []Message
	Temperature float64
	MaxTokens   int
}

type Response struct {
	Answer string `json:"answer"`
	Model  string `json:"model"`
}

type Provider interface {
	Generate(ctx context.Context, req Request) (Response, error)
	Name() string
	Model() string
}

func NewProvider(name, model string) (Provider, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if model == "" {
		model = "local-rag"
	}
	switch name {
	case "", "local":
		return LocalProvider{model: model}, nil
	default:
		return nil, errors.New("unsupported generation provider")
	}
}

type LocalProvider struct {
	model string
}

func (p LocalProvider) Name() string {
	return "local"
}

func (p LocalProvider) Model() string {
	return p.model
}

func (p LocalProvider) Generate(_ context.Context, req Request) (Response, error) {
	var user string
	var contextBlock string
	for _, msg := range req.Messages {
		switch msg.Role {
		case "user":
			user = strings.TrimSpace(msg.Content)
		case "system":
			contextBlock = strings.TrimSpace(msg.Content)
		}
	}
	answer := "未检索到足够上下文，无法回答。"
	if contextBlock != "" {
		answer = "基于已检索知识库上下文的本地回答草稿：" + summarizeContext(contextBlock)
	}
	if user != "" {
		answer += "\n问题：" + user
	}
	return Response{Answer: answer, Model: p.model}, nil
}

func summarizeContext(text string) string {
	const max = 600
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= max {
		return text
	}
	return text[:max] + "..."
}
