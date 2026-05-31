package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
)

// MockProvider 是内置的开发/测试渠道,沿用既有的 JSON + HMAC-SHA256 回调语义:
//   - 未配置密钥时保留开发体验(任何回调都视为合法);
//   - 配置密钥后,异步通知必须在 X-Payment-Signature 头携带 "sha256=<hex>"。
type MockProvider struct {
	Secret string
}

// NewMockProvider 构造 mock 渠道。secret 为空表示开发模式。
func NewMockProvider(secret string) MockProvider {
	return MockProvider{Secret: strings.TrimSpace(secret)}
}

// Channel 返回渠道编码。
func (p MockProvider) Channel() string { return "mock" }

// CreatePrepay 对 mock 渠道不产生真实跳转,仅返回中文提示。
func (p MockProvider) CreatePrepay(_ context.Context, in PrepayInput) (Prepay, error) {
	return Prepay{
		Channel: "mock",
		Method:  "mock",
		Message: "mock 渠道无需真实支付,可直接发送 mock 回调将支付单标记为已支付。",
		Extra: map[string]any{
			"pay_no": in.PayNo,
		},
	}, nil
}

// VerifyNotify 校验 HMAC 签名并解析 JSON 回调体。
func (p MockProvider) VerifyNotify(_ context.Context, n Notify) (NotifyResult, error) {
	if !verifyHMACSignature(p.Secret, n.Body, headerValue(n.Header, "X-Payment-Signature")) {
		return NotifyResult{}, ErrInvalidSignature
	}
	var body struct {
		PayNo         string         `json:"pay_no"`
		TransactionID string         `json:"transaction_id"`
		Status        string         `json:"status"`
		Metadata      map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(n.Body, &body); err != nil {
		return NotifyResult{}, ErrInvalidNotify
	}
	payNo := strings.TrimSpace(body.PayNo)
	if payNo == "" {
		return NotifyResult{}, ErrInvalidNotify
	}
	status := strings.ToLower(strings.TrimSpace(body.Status))
	if status == "" {
		status = "paid"
	}
	return NotifyResult{
		PayNo:         payNo,
		TransactionID: strings.TrimSpace(body.TransactionID),
		Status:        status,
		Raw:           body.Metadata,
	}, nil
}

// NotifyResponse 返回 mock 渠道的 JSON 应答。
func (p MockProvider) NotifyResponse(ok bool) (int, string, string) {
	if ok {
		return http.StatusOK, "application/json; charset=utf-8", `{"code":0,"message":"ok"}`
	}
	return http.StatusBadRequest, "application/json; charset=utf-8", `{"code":1,"message":"failed"}`
}

// verifyHMACSignature 复用既有支付回调验签逻辑:空密钥放行,否则比对 HMAC-SHA256。
func verifyHMACSignature(secret string, body []byte, signature string) bool {
	if strings.TrimSpace(secret) == "" {
		return true
	}
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
