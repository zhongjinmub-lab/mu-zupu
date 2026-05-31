package channel

import (
	"errors"
	"strings"
	"time"
)

// 渠道状态常量。
const (
	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
	StatusArchived = "archived"
)

// ChannelType 表示一种内置渠道类型定义。
type ChannelType struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Installable bool   `json:"installable"`
}

// Channel 表示一个持久化的 Agent 渠道接入点。
type Channel struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	AgentID    string         `json:"agent_id"`
	Type       string         `json:"type"`
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	ChannelKey string         `json:"channel_key"`
	Config     map[string]any `json:"config"`
	CreatedBy  string         `json:"created_by,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// CreateChannelRequest 是创建渠道接入点的请求体。
type CreateChannelRequest struct {
	AgentID string         `json:"agent_id" binding:"required"`
	Type    string         `json:"type" binding:"required"`
	Name    string         `json:"name" binding:"required"`
	Config  map[string]any `json:"config"`
}

// Normalize 归一化创建请求字段。
func (r *CreateChannelRequest) Normalize() {
	r.AgentID = strings.TrimSpace(r.AgentID)
	r.Type = strings.ToLower(strings.TrimSpace(r.Type))
	r.Name = strings.TrimSpace(r.Name)
}

// Validate 校验创建请求：agent_id、name 必填，type 必须是内置且可用的渠道类型。
func (r CreateChannelRequest) Validate() error {
	if r.AgentID == "" {
		return errors.New("agent_id is required")
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	if len([]rune(r.Name)) > 128 {
		return errors.New("name must be at most 128 characters")
	}
	def, ok := FindChannelType(r.Type)
	if !ok {
		return errors.New("unknown channel type")
	}
	if !def.Installable {
		return errors.New("channel type is not available yet")
	}
	return nil
}

// DefaultChannelTypes 返回内置渠道类型目录。active 类型可创建，planned 类型暂不可用。
func DefaultChannelTypes() []ChannelType {
	defs := []ChannelType{
		{Type: "web", Name: "网页嵌入", Category: "web", Status: "active", Description: "在网站中嵌入对话组件，通过 channel_key 接入。"},
		{Type: "h5", Name: "移动 H5", Category: "web", Status: "active", Description: "面向移动端浏览器的 H5 对话页面。"},
		{Type: "api", Name: "开放 API", Category: "api", Status: "active", Description: "通过 channel_key 以 API 方式接入已发布 Agent。"},
		{Type: "wechat_official", Name: "微信公众号", Category: "social", Status: "planned", Description: "对接微信公众号，需后续补充服务号配置与回调验签。"},
		{Type: "wechat_work", Name: "企业微信", Category: "social", Status: "planned", Description: "对接企业微信应用，需后续补充企业凭据与回调。"},
	}
	for i := range defs {
		defs[i].Installable = defs[i].Status == "active"
	}
	return defs
}

// FindChannelType 按类型编码查找内置渠道类型（忽略大小写和空白）。
func FindChannelType(t string) (ChannelType, bool) {
	t = strings.ToLower(strings.TrimSpace(t))
	for _, def := range DefaultChannelTypes() {
		if def.Type == t {
			return def, true
		}
	}
	return ChannelType{}, false
}
