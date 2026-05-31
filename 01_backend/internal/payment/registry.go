package payment

import (
	"fmt"
	"strings"
)

// Registry 管理一组已启用的支付渠道,按渠道编码查找 Provider。
type Registry struct {
	providers map[string]Provider
	order     []string
}

// NewRegistry 用给定的 Provider 列表构造注册表,后注册的同名渠道会覆盖先注册的。
func NewRegistry(providers ...Provider) *Registry {
	r := &Registry{providers: make(map[string]Provider, len(providers))}
	for _, p := range providers {
		if p == nil {
			continue
		}
		channel := normalizeChannel(p.Channel())
		if channel == "" {
			continue
		}
		if _, exists := r.providers[channel]; !exists {
			r.order = append(r.order, channel)
		}
		r.providers[channel] = p
	}
	return r
}

// Get 返回指定渠道的 Provider。
func (r *Registry) Get(channel string) (Provider, bool) {
	if r == nil {
		return nil, false
	}
	p, ok := r.providers[normalizeChannel(channel)]
	return p, ok
}

// IsEnabled 判断渠道是否已启用。
func (r *Registry) IsEnabled(channel string) bool {
	_, ok := r.Get(channel)
	return ok
}

// Channels 返回已启用渠道编码列表(保持注册顺序)。
func (r *Registry) Channels() []string {
	if r == nil {
		return nil
	}
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// RegistryConfig 是从应用配置构建注册表所需的参数。
type RegistryConfig struct {
	Channels   string // 逗号分隔的启用渠道,例如 "mock,alipay,wechat";为空时默认仅 mock
	MockSecret string // mock 渠道的 HMAC 验签密钥(可为空,开发模式)
	Alipay     AlipayConfig
	Wechat     WechatConfig
}

// BuildRegistry 根据配置构建注册表。未知渠道或渠道配置非法时返回错误。
func BuildRegistry(cfg RegistryConfig) (*Registry, error) {
	names := splitChannels(cfg.Channels)
	if len(names) == 0 {
		names = []string{"mock"}
	}
	providers := make([]Provider, 0, len(names))
	for _, name := range names {
		switch name {
		case "mock":
			providers = append(providers, NewMockProvider(cfg.MockSecret))
		case "alipay":
			ap, err := NewAlipayProvider(cfg.Alipay)
			if err != nil {
				return nil, fmt.Errorf("payment channel alipay: %w", err)
			}
			providers = append(providers, ap)
		case "wechat":
			wp, err := NewWechatProvider(cfg.Wechat)
			if err != nil {
				return nil, fmt.Errorf("payment channel wechat: %w", err)
			}
			providers = append(providers, wp)
		default:
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedChannel, name)
		}
	}
	return NewRegistry(providers...), nil
}

func splitChannels(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		channel := normalizeChannel(part)
		if channel == "" {
			continue
		}
		if _, ok := seen[channel]; ok {
			continue
		}
		seen[channel] = struct{}{}
		out = append(out, channel)
	}
	return out
}

func normalizeChannel(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
