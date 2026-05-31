package payment

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// alipayDefaultGateway 是支付宝正式环境网关。
const alipayDefaultGateway = "https://openapi.alipay.com/gateway.do"

// AlipayConfig 是支付宝渠道配置。
type AlipayConfig struct {
	AppID      string // 支付宝开放平台应用 App ID
	PrivateKey string // 应用私钥(PKCS1/PKCS8 PEM 或裸 base64),用于请求签名
	PublicKey  string // 支付宝公钥(PKIX/PKCS1 PEM 或裸 base64),用于异步通知验签
	Gateway    string // 网关地址,默认正式环境
	SignType   string // 签名算法,仅支持 RSA2
}

// IsConfigured 判断是否提供了支付宝渠道所需的最小配置。
func (c AlipayConfig) IsConfigured() bool {
	return strings.TrimSpace(c.AppID) != "" &&
		strings.TrimSpace(c.PrivateKey) != "" &&
		strings.TrimSpace(c.PublicKey) != ""
}

// AlipayProvider 实现支付宝电脑网站支付(alipay.trade.page.pay)。
//
// CreatePrepay 仅在本地构造并签名跳转 URL,不发起网络请求;
// VerifyNotify 解析 application/x-www-form-urlencoded 异步通知并用支付宝公钥 RSA2 验签。
type AlipayProvider struct {
	appID    string
	gateway  string
	signType string
	priv     *rsa.PrivateKey
	pub      *rsa.PublicKey
}

// NewAlipayProvider 构造支付宝渠道。缺失配置或密钥非法时返回错误。
func NewAlipayProvider(cfg AlipayConfig) (AlipayProvider, error) {
	appID := strings.TrimSpace(cfg.AppID)
	if appID == "" {
		return AlipayProvider{}, errors.New("alipay app_id is required")
	}
	signType := strings.ToUpper(strings.TrimSpace(cfg.SignType))
	if signType == "" {
		signType = "RSA2"
	}
	if signType != "RSA2" {
		return AlipayProvider{}, errors.New("alipay only supports RSA2 sign type")
	}
	priv, err := parseRSAPrivateKey(cfg.PrivateKey)
	if err != nil {
		return AlipayProvider{}, fmt.Errorf("alipay private key: %w", err)
	}
	pub, err := parseRSAPublicKey(cfg.PublicKey)
	if err != nil {
		return AlipayProvider{}, fmt.Errorf("alipay public key: %w", err)
	}
	gateway := strings.TrimSpace(cfg.Gateway)
	if gateway == "" {
		gateway = alipayDefaultGateway
	}
	return AlipayProvider{
		appID:    appID,
		gateway:  gateway,
		signType: signType,
		priv:     priv,
		pub:      pub,
	}, nil
}

// Channel 返回渠道编码。
func (p AlipayProvider) Channel() string { return "alipay" }

// CreatePrepay 构造已签名的支付宝跳转支付 URL。
func (p AlipayProvider) CreatePrepay(_ context.Context, in PrepayInput) (Prepay, error) {
	if strings.TrimSpace(in.PayNo) == "" {
		return Prepay{}, errors.New("pay_no is required")
	}
	subject := strings.TrimSpace(in.Subject)
	if subject == "" {
		subject = "订单支付 " + in.PayNo
	}
	bizContent := fmt.Sprintf(
		`{"out_trade_no":%q,"total_amount":%q,"subject":%q,"product_code":"FAST_INSTANT_TRADE_PAY"}`,
		in.PayNo, formatYuan(in.AmountCents), subject,
	)
	params := map[string]string{
		"app_id":      p.appID,
		"method":      "alipay.trade.page.pay",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   p.signType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": bizContent,
	}
	if u := strings.TrimSpace(in.NotifyURL); u != "" {
		params["notify_url"] = u
	}
	if u := strings.TrimSpace(in.ReturnURL); u != "" {
		params["return_url"] = u
	}
	// 请求签名:保留 sign_type 参与签名。
	sign, err := signRSA2(p.priv, buildSignContent(params, false))
	if err != nil {
		return Prepay{}, err
	}
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}
	query.Set("sign", sign)
	return Prepay{
		Channel: "alipay",
		Method:  "redirect",
		PayURL:  p.gateway + "?" + query.Encode(),
		Message: "请在新页面跳转到支付宝完成支付。",
	}, nil
}

// VerifyNotify 解析并验签支付宝异步通知。
func (p AlipayProvider) VerifyNotify(_ context.Context, n Notify) (NotifyResult, error) {
	values, err := url.ParseQuery(string(n.Body))
	if err != nil {
		return NotifyResult{}, ErrInvalidNotify
	}
	params := make(map[string]string, len(values))
	for key := range values {
		params[key] = values.Get(key)
	}
	sign := strings.TrimSpace(params["sign"])
	if sign == "" {
		return NotifyResult{}, ErrInvalidSignature
	}
	// 异步通知验签:排除 sign 与 sign_type。
	if err := verifyRSA2(p.pub, buildSignContent(params, true), sign); err != nil {
		return NotifyResult{}, ErrInvalidSignature
	}
	payNo := strings.TrimSpace(params["out_trade_no"])
	if payNo == "" {
		return NotifyResult{}, ErrInvalidNotify
	}
	status := mapAlipayTradeStatus(params["trade_status"])
	if status == "" {
		return NotifyResult{}, fmt.Errorf("%w: trade_status=%s", ErrInvalidNotify, params["trade_status"])
	}
	raw := make(map[string]any, len(params))
	for k, v := range params {
		if k == "sign" {
			continue
		}
		raw[k] = v
	}
	return NotifyResult{
		PayNo:         payNo,
		TransactionID: strings.TrimSpace(params["trade_no"]),
		Status:        status,
		Raw:           raw,
	}, nil
}

// NotifyResponse 返回支付宝期望的纯文本应答。
func (p AlipayProvider) NotifyResponse(ok bool) (int, string, string) {
	if ok {
		return http.StatusOK, "text/plain; charset=utf-8", "success"
	}
	return http.StatusOK, "text/plain; charset=utf-8", "failure"
}

// mapAlipayTradeStatus 将支付宝 trade_status 映射为内部支付状态。
func mapAlipayTradeStatus(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		return "paid"
	case "TRADE_CLOSED":
		return "failed"
	default:
		// WAIT_BUYER_PAY 等中间态不更新支付单。
		return ""
	}
}

// formatYuan 将金额(分)格式化为支付宝要求的元字符串(两位小数)。
func formatYuan(cents int64) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}
	out := fmt.Sprintf("%d.%02d", cents/100, cents%100)
	if negative {
		return "-" + out
	}
	return out
}
