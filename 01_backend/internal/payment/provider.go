// Package payment 提供可插拔的第三方支付渠道抽象。
//
// 该包刻意只依赖 Go 标准库,便于在没有数据库/Web 框架依赖的环境下独立单测。
// 现有 mock 渠道的 JSON + HMAC-SHA256 回调语义被保留为内置 MockProvider,
// 真实渠道(如支付宝)通过实现 Provider 接口接入。
package payment

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// PrepayInput 是创建第三方预支付单所需的标准化输入。
type PrepayInput struct {
	PayNo       string // 商户支付单号(payment_orders.pay_no),作为第三方 out_trade_no
	AmountCents int64  // 金额(分)
	Currency    string // 币种,例如 CNY
	Subject     string // 订单标题/描述
	NotifyURL   string // 异步通知回调地址(服务端可访问)
	ReturnURL   string // 同步跳转地址(用户浏览器)
}

// Prepay 是创建预支付单的结果,供前端引导用户完成支付。
type Prepay struct {
	Channel   string         `json:"channel"`
	Method    string         `json:"method"`               // redirect / qrcode / mock
	PayURL    string         `json:"pay_url,omitempty"`    // 跳转支付地址
	QRContent string         `json:"qr_content,omitempty"` // 二维码内容
	Message   string         `json:"message,omitempty"`    // 中文提示
	Extra     map[string]any `json:"extra,omitempty"`      // 渠道附加信息(不含敏感明文)
}

// Notify 是第三方异步通知的原始请求。
type Notify struct {
	Header      http.Header
	Body        []byte
	ContentType string
}

// NotifyResult 是验签并解析后的标准化通知结果。
type NotifyResult struct {
	PayNo         string         // 商户支付单号
	TransactionID string         // 第三方交易号
	Status        string         // paid / failed
	Raw           map[string]any // 渠道原始字段摘要(用于审计)
}

// 主动查单返回的标准化状态。
const (
	QueryStatusPaid     = "paid"
	QueryStatusFailed   = "failed"
	QueryStatusPending  = "pending"
	QueryStatusNotFound = "not_found"
)

// QueryResult 是向渠道主动查单后的标准化结果。
type QueryResult struct {
	PayNo         string
	TransactionID string
	Status        string // paid / failed / pending / not_found
	Raw           map[string]any
}

// OrderQuerier 是可选能力接口:支持向渠道主动查单对账的渠道实现它。
// mock 等不支持远程查单的渠道无需实现,上层据此回退到本地状态。
type OrderQuerier interface {
	// QueryOrder 按商户支付单号向渠道查询订单真实状态。
	QueryOrder(ctx context.Context, payNo string) (QueryResult, error)
}

// Provider 抽象一个支付渠道。所有实现都必须是无状态、并发安全的。
type Provider interface {
	// Channel 返回渠道编码,例如 "mock" 或 "alipay"。
	Channel() string
	// CreatePrepay 创建预支付单,返回用于引导用户支付的信息。
	CreatePrepay(ctx context.Context, in PrepayInput) (Prepay, error)
	// VerifyNotify 校验并解析第三方异步通知,返回标准化结果。
	VerifyNotify(ctx context.Context, n Notify) (NotifyResult, error)
	// NotifyResponse 返回应答第三方所需的 HTTP 状态码、Content-Type 与响应体。
	NotifyResponse(ok bool) (status int, contentType, body string)
}

// 包级错误,便于上层做分支处理。
var (
	ErrUnsupportedChannel = errors.New("unsupported payment channel")
	ErrInvalidSignature   = errors.New("payment notify signature invalid")
	ErrInvalidNotify      = errors.New("payment notify payload invalid")
)

func headerValue(h http.Header, key string) string {
	if h == nil {
		return ""
	}
	return strings.TrimSpace(h.Get(key))
}
