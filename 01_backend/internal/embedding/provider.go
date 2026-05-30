package embedding

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"math"
	"strings"
)

const Dim = 1536

type Provider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Name() string
	Model() string
}

func NewProvider(name, model string) (Provider, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if model == "" {
		model = "local-hash-1536"
	}
	switch name {
	case "", "local":
		return LocalProvider{model: model}, nil
	default:
		return nil, errors.New("unsupported embedding provider")
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

func (p LocalProvider) Embed(_ context.Context, text string) ([]float32, error) {
	out := make([]float32, Dim)
	if text == "" {
		return out, nil
	}
	seed := sha256.Sum256([]byte(text))
	for i := range out {
		block := sha256.Sum256(append(seed[:], byte(i), byte(i>>8)))
		v := binary.BigEndian.Uint32(block[:4])
		out[i] = float32(float64(v)/float64(math.MaxUint32)*2 - 1)
	}
	normalize(out)
	return out, nil
}

func normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x * x)
	}
	if sum == 0 {
		return
	}
	norm := float32(math.Sqrt(sum))
	for i := range v {
		v[i] = v[i] / norm
	}
}
