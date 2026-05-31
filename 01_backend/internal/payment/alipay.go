package payment

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	HTTPClient *http.Client
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
	appID      string
	gateway    string
	signType   string
	priv       *rsa.PrivateKey
	pub        *rsa.PublicKey
	httpClient *http.Client
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
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return AlipayProvider{
		appID:      appID,
		gateway:    gateway,
		signType:   signType,
		priv:       priv,
		pub:        pub,
		httpClient: client,
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

// QueryOrder 调用 alipay.trade.query 查询订单真实状态。
func (p AlipayProvider) QueryOrder(ctx context.Context, payNo string) (QueryResult, error) {
	payNo = strings.TrimSpace(payNo)
	if payNo == "" {
		return QueryResult{}, errors.New("pay_no is required")
	}
	params := map[string]string{
		"app_id":      p.appID,
		"method":      "alipay.trade.query",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   p.signType,
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": fmt.Sprintf(`{"out_trade_no":%q}`, payNo),
	}
	sign, err := signRSA2(p.priv, buildSignContent(params, false))
	if err != nil {
		return QueryResult{}, err
	}
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	form.Set("sign", sign)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.gateway, strings.NewReader(form.Encode()))
	if err != nil {
		return QueryResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return QueryResult{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return QueryResult{}, err
	}

	rawNode, ok := extractJSONObjectNode(body, "alipay_trade_query_response")
	if !ok {
		return QueryResult{}, ErrInvalidNotify
	}
	var outer struct {
		Sign string `json:"sign"`
	}
	_ = json.Unmarshal(body, &outer)
	if strings.TrimSpace(outer.Sign) != "" {
		if err := verifyRSA2(p.pub, rawNode, outer.Sign); err != nil {
			return QueryResult{}, ErrInvalidSignature
		}
	}
	var content struct {
		Code        string `json:"code"`
		SubCode     string `json:"sub_code"`
		TradeNo     string `json:"trade_no"`
		OutTradeNo  string `json:"out_trade_no"`
		TradeStatus string `json:"trade_status"`
	}
	if err := json.Unmarshal([]byte(rawNode), &content); err != nil {
		return QueryResult{}, ErrInvalidNotify
	}

	raw := map[string]any{
		"code":         content.Code,
		"trade_status": content.TradeStatus,
		"trade_no":     content.TradeNo,
	}
	if content.SubCode != "" {
		raw["sub_code"] = content.SubCode
	}
	result := QueryResult{
		PayNo:         payNo,
		TransactionID: strings.TrimSpace(content.TradeNo),
		Raw:           raw,
	}
	if content.Code != "10000" {
		// 订单不存在按未找到处理,其余业务错误按待支付处理。
		if strings.EqualFold(content.SubCode, "ACQ.TRADE_NOT_EXIST") {
			result.Status = QueryStatusNotFound
		} else {
			result.Status = QueryStatusPending
		}
		return result, nil
	}
	result.Status = mapAlipayQueryStatus(content.TradeStatus)
	return result, nil
}

// mapAlipayQueryStatus 将查单的 trade_status 映射为内部状态(含待支付)。
func mapAlipayQueryStatus(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		return QueryStatusPaid
	case "TRADE_CLOSED":
		return QueryStatusFailed
	default:
		// WAIT_BUYER_PAY 等视为待支付。
		return QueryStatusPending
	}
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
