package payment

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// wechatDefaultGateway 是微信支付 v3 正式环境网关。
const wechatDefaultGateway = "https://api.mch.weixin.qq.com"

// WechatConfig 是微信支付 v3 渠道配置。
type WechatConfig struct {
	AppID          string // 公众号/小程序/App 的 AppID
	MchID          string // 商户号
	SerialNo       string // 商户 API 证书序列号
	APIv3Key       string // APIv3 密钥(32 字节),用于回调资源体 AES-256-GCM 解密
	PrivateKey     string // 商户 API 私钥(PKCS1/PKCS8 PEM 或裸 base64),用于请求签名
	PlatformPublic string // 微信支付平台证书公钥(PKIX/PKCS1 PEM 或裸 base64),用于回调验签
	Gateway        string // 网关地址,默认正式环境
	NotifyURL      string // 兜底回调地址(创建预支付时若未显式传入则使用)
	HTTPClient     *http.Client
}

// IsConfigured 判断是否提供了微信渠道所需的最小配置。
func (c WechatConfig) IsConfigured() bool {
	return strings.TrimSpace(c.AppID) != "" &&
		strings.TrimSpace(c.MchID) != "" &&
		strings.TrimSpace(c.SerialNo) != "" &&
		len(strings.TrimSpace(c.APIv3Key)) == 32 &&
		strings.TrimSpace(c.PrivateKey) != "" &&
		strings.TrimSpace(c.PlatformPublic) != ""
}

// WechatProvider 实现微信支付 v3 Native 下单(扫码支付)。
//
// CreatePrepay 通过 HTTP 调用微信网关下单并返回 code_url(二维码内容);
// VerifyNotify 校验微信回调签名并用 APIv3 密钥 AES-256-GCM 解密资源体。
type WechatProvider struct {
	appID       string
	mchID       string
	serialNo    string
	apiV3Key    []byte
	priv        *rsa.PrivateKey
	platformPub *rsa.PublicKey
	gateway     string
	notifyURL   string
	httpClient  *http.Client
}

// NewWechatProvider 构造微信支付渠道。缺失配置或密钥非法时返回错误。
func NewWechatProvider(cfg WechatConfig) (WechatProvider, error) {
	appID := strings.TrimSpace(cfg.AppID)
	mchID := strings.TrimSpace(cfg.MchID)
	serialNo := strings.TrimSpace(cfg.SerialNo)
	apiV3Key := strings.TrimSpace(cfg.APIv3Key)
	if appID == "" || mchID == "" || serialNo == "" {
		return WechatProvider{}, errors.New("wechat app_id, mch_id and serial_no are required")
	}
	if len(apiV3Key) != 32 {
		return WechatProvider{}, errors.New("wechat api_v3_key must be 32 bytes")
	}
	priv, err := parseRSAPrivateKey(cfg.PrivateKey)
	if err != nil {
		return WechatProvider{}, fmt.Errorf("wechat private key: %w", err)
	}
	platformPub, err := parseRSAPublicKey(cfg.PlatformPublic)
	if err != nil {
		return WechatProvider{}, fmt.Errorf("wechat platform public key: %w", err)
	}
	gateway := strings.TrimRight(strings.TrimSpace(cfg.Gateway), "/")
	if gateway == "" {
		gateway = wechatDefaultGateway
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return WechatProvider{
		appID:       appID,
		mchID:       mchID,
		serialNo:    serialNo,
		apiV3Key:    []byte(apiV3Key),
		priv:        priv,
		platformPub: platformPub,
		gateway:     gateway,
		notifyURL:   strings.TrimSpace(cfg.NotifyURL),
		httpClient:  client,
	}, nil
}

// Channel 返回渠道编码。
func (p WechatProvider) Channel() string { return "wechat" }

// CreatePrepay 调用微信 Native 下单并返回二维码内容(code_url)。
func (p WechatProvider) CreatePrepay(ctx context.Context, in PrepayInput) (Prepay, error) {
	if strings.TrimSpace(in.PayNo) == "" {
		return Prepay{}, errors.New("pay_no is required")
	}
	notifyURL := strings.TrimSpace(in.NotifyURL)
	if notifyURL == "" {
		notifyURL = p.notifyURL
	}
	if notifyURL == "" {
		return Prepay{}, errors.New("wechat notify_url is required")
	}
	subject := strings.TrimSpace(in.Subject)
	if subject == "" {
		subject = "订单支付 " + in.PayNo
	}
	currency := strings.ToUpper(strings.TrimSpace(in.Currency))
	if currency == "" {
		currency = "CNY"
	}
	reqBody := map[string]any{
		"appid":        p.appID,
		"mchid":        p.mchID,
		"description":  subject,
		"out_trade_no": in.PayNo,
		"notify_url":   notifyURL,
		"amount": map[string]any{
			"total":    in.AmountCents,
			"currency": currency,
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return Prepay{}, err
	}

	const path = "/v3/pay/transactions/native"
	authorization, err := p.buildAuthorization(http.MethodPost, path, bodyBytes)
	if err != nil {
		return Prepay{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.gateway+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return Prepay{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", authorization)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return Prepay{}, err
	}
	defer resp.Body.Close()
	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Prepay{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return Prepay{}, fmt.Errorf("wechat native order failed: status=%d body=%s", resp.StatusCode, string(respBytes))
	}
	var parsed struct {
		CodeURL string `json:"code_url"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return Prepay{}, ErrInvalidNotify
	}
	if parsed.CodeURL == "" {
		return Prepay{}, errors.New("wechat native order missing code_url")
	}
	return Prepay{
		Channel:   "wechat",
		Method:    "qrcode",
		QRContent: parsed.CodeURL,
		Message:   "请使用微信扫一扫完成支付。",
	}, nil
}

// VerifyNotify 校验微信回调签名并解密资源体。
func (p WechatProvider) VerifyNotify(_ context.Context, n Notify) (NotifyResult, error) {
	timestamp := headerValue(n.Header, "Wechatpay-Timestamp")
	nonce := headerValue(n.Header, "Wechatpay-Nonce")
	signature := headerValue(n.Header, "Wechatpay-Signature")
	if timestamp == "" || nonce == "" || signature == "" {
		return NotifyResult{}, ErrInvalidSignature
	}
	content := timestamp + "\n" + nonce + "\n" + string(n.Body) + "\n"
	if err := verifyRSA2(p.platformPub, content, signature); err != nil {
		return NotifyResult{}, ErrInvalidSignature
	}

	var envelope struct {
		Resource struct {
			Algorithm      string `json:"algorithm"`
			Ciphertext     string `json:"ciphertext"`
			Nonce          string `json:"nonce"`
			AssociatedData string `json:"associated_data"`
		} `json:"resource"`
	}
	if err := json.Unmarshal(n.Body, &envelope); err != nil {
		return NotifyResult{}, ErrInvalidNotify
	}
	if !strings.EqualFold(envelope.Resource.Algorithm, "AEAD_AES_256_GCM") {
		return NotifyResult{}, fmt.Errorf("%w: unsupported algorithm %s", ErrInvalidNotify, envelope.Resource.Algorithm)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Resource.Ciphertext)
	if err != nil {
		return NotifyResult{}, ErrInvalidNotify
	}
	plaintext, err := decryptAESGCM(p.apiV3Key, []byte(envelope.Resource.Nonce), ciphertext, []byte(envelope.Resource.AssociatedData))
	if err != nil {
		return NotifyResult{}, err
	}
	var resource struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		TradeState    string `json:"trade_state"`
	}
	if err := json.Unmarshal(plaintext, &resource); err != nil {
		return NotifyResult{}, ErrInvalidNotify
	}
	payNo := strings.TrimSpace(resource.OutTradeNo)
	if payNo == "" {
		return NotifyResult{}, ErrInvalidNotify
	}
	status := mapWechatTradeState(resource.TradeState)
	if status == "" {
		return NotifyResult{}, fmt.Errorf("%w: trade_state=%s", ErrInvalidNotify, resource.TradeState)
	}
	return NotifyResult{
		PayNo:         payNo,
		TransactionID: strings.TrimSpace(resource.TransactionID),
		Status:        status,
		Raw: map[string]any{
			"out_trade_no":   resource.OutTradeNo,
			"transaction_id": resource.TransactionID,
			"trade_state":    resource.TradeState,
		},
	}, nil
}

// QueryOrder 调用微信查单接口(按商户订单号)查询订单真实状态。
func (p WechatProvider) QueryOrder(ctx context.Context, payNo string) (QueryResult, error) {
	payNo = strings.TrimSpace(payNo)
	if payNo == "" {
		return QueryResult{}, errors.New("pay_no is required")
	}
	path := "/v3/pay/transactions/out-trade-no/" + url.PathEscape(payNo) + "?mchid=" + url.QueryEscape(p.mchID)
	authorization, err := p.buildAuthorization(http.MethodGet, path, nil)
	if err != nil {
		return QueryResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.gateway+path, nil)
	if err != nil {
		return QueryResult{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", authorization)
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return QueryResult{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return QueryResult{}, err
	}
	if resp.StatusCode == http.StatusNotFound {
		return QueryResult{PayNo: payNo, Status: QueryStatusNotFound}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return QueryResult{}, fmt.Errorf("wechat query failed: status=%d body=%s", resp.StatusCode, string(body))
	}
	// 若应答带签名头则验签(应答由微信平台证书签名)。
	ts := resp.Header.Get("Wechatpay-Timestamp")
	nonce := resp.Header.Get("Wechatpay-Nonce")
	sig := resp.Header.Get("Wechatpay-Signature")
	if ts != "" && nonce != "" && sig != "" {
		if err := verifyRSA2(p.platformPub, ts+"\n"+nonce+"\n"+string(body)+"\n", sig); err != nil {
			return QueryResult{}, ErrInvalidSignature
		}
	}
	var content struct {
		OutTradeNo    string `json:"out_trade_no"`
		TransactionID string `json:"transaction_id"`
		TradeState    string `json:"trade_state"`
	}
	if err := json.Unmarshal(body, &content); err != nil {
		return QueryResult{}, ErrInvalidNotify
	}
	return QueryResult{
		PayNo:         payNo,
		TransactionID: strings.TrimSpace(content.TransactionID),
		Status:        mapWechatQueryState(content.TradeState),
		Raw: map[string]any{
			"trade_state":    content.TradeState,
			"transaction_id": content.TransactionID,
		},
	}, nil
}

// mapWechatQueryState 将查单的 trade_state 映射为内部状态(含待支付)。
func mapWechatQueryState(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "SUCCESS":
		return QueryStatusPaid
	case "CLOSED", "REVOKED", "PAYERROR":
		return QueryStatusFailed
	default:
		// NOTPAY / USERPAYING / REFUND / ACCEPT 等视为待支付。
		return QueryStatusPending
	}
}

// NotifyResponse 返回微信期望的 JSON 应答。
func (p WechatProvider) NotifyResponse(ok bool) (int, string, string) {
	if ok {
		return http.StatusOK, "application/json; charset=utf-8", `{"code":"SUCCESS","message":"成功"}`
	}
	return http.StatusInternalServerError, "application/json; charset=utf-8", `{"code":"FAIL","message":"失败"}`
}

// buildAuthorization 构造微信支付 v3 请求签名的 Authorization 头。
func (p WechatProvider) buildAuthorization(method, path string, body []byte) (string, error) {
	nonce, err := randomNonce(16)
	if err != nil {
		return "", err
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signContent := method + "\n" + path + "\n" + timestamp + "\n" + nonce + "\n" + string(body) + "\n"
	signature, err := signRSA2(p.priv, signContent)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		`WECHATPAY2-SHA256-RSA2048 mchid="%s",nonce_str="%s",timestamp="%s",serial_no="%s",signature="%s"`,
		p.mchID, nonce, timestamp, p.serialNo, signature,
	), nil
}

// mapWechatTradeState 将微信 trade_state 映射为内部支付状态。
func mapWechatTradeState(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "SUCCESS":
		return "paid"
	case "CLOSED", "REVOKED", "PAYERROR":
		return "failed"
	default:
		// NOTPAY / USERPAYING / REFUND 等不更新支付单。
		return ""
	}
}

func randomNonce(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
