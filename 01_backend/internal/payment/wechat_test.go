package payment

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testAPIv3Key = "0123456789abcdef0123456789abcdef" // 32 字节

func newTestWechatProvider(t *testing.T, gateway string, client *http.Client) WechatProvider {
	t.Helper()
	privPEM, pubPEM := mustAlipayKeys(t)
	provider, err := NewWechatProvider(WechatConfig{
		AppID:          "wxappid",
		MchID:          "1900000001",
		SerialNo:       "SERIAL123",
		APIv3Key:       testAPIv3Key,
		PrivateKey:     privPEM,
		PlatformPublic: pubPEM,
		Gateway:        gateway,
		HTTPClient:     client,
	})
	if err != nil {
		t.Fatalf("new wechat provider: %v", err)
	}
	return provider
}

func TestWechatCreatePrepayReturnsCodeURL(t *testing.T) {
	const codeURL = "weixin://wxpay/bizpayurl?pr=abc123"
	var gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code_url":"` + codeURL + `","prepay_id":"wx123"}`))
	}))
	defer server.Close()

	provider := newTestWechatProvider(t, server.URL, server.Client())
	prepay, err := provider.CreatePrepay(context.Background(), PrepayInput{
		PayNo:       "PO20260531",
		AmountCents: 12345,
		Currency:    "CNY",
		Subject:     "套餐订阅",
		NotifyURL:   "https://api.example.com/api/v1/payment-notify/wechat",
	})
	if err != nil {
		t.Fatalf("create prepay: %v", err)
	}
	if prepay.Method != "qrcode" || prepay.QRContent != codeURL || prepay.Channel != "wechat" {
		t.Fatalf("unexpected prepay: %#v", prepay)
	}
	if !strings.HasPrefix(gotAuth, "WECHATPAY2-SHA256-RSA2048 ") {
		t.Fatalf("unexpected authorization header: %q", gotAuth)
	}
	for _, field := range []string{`mchid="1900000001"`, `serial_no="SERIAL123"`, `signature="`, `nonce_str="`, `timestamp="`} {
		if !strings.Contains(gotAuth, field) {
			t.Fatalf("authorization missing %s: %q", field, gotAuth)
		}
	}
	if gotBody["out_trade_no"] != "PO20260531" {
		t.Fatalf("request body missing out_trade_no: %#v", gotBody)
	}
	amount, _ := gotBody["amount"].(map[string]any)
	if amount == nil || amount["total"].(float64) != 12345 {
		t.Fatalf("request body amount mismatch: %#v", gotBody)
	}
}

func TestWechatCreatePrepayRequiresNotifyURL(t *testing.T) {
	provider := newTestWechatProvider(t, "https://api.mch.weixin.qq.com", &http.Client{})
	if _, err := provider.CreatePrepay(context.Background(), PrepayInput{PayNo: "PO1", AmountCents: 100}); err == nil {
		t.Fatal("expected error when notify_url is missing")
	}
}

func TestWechatCreatePrepayPropagatesGatewayError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"PARAM_ERROR","message":"invalid"}`))
	}))
	defer server.Close()
	provider := newTestWechatProvider(t, server.URL, server.Client())
	if _, err := provider.CreatePrepay(context.Background(), PrepayInput{
		PayNo:     "PO1",
		NotifyURL: "https://api.example.com/api/v1/payment-notify/wechat",
	}); err == nil {
		t.Fatal("expected error for non-200 gateway response")
	}
}

// buildWechatNotify 构造一份合法的微信回调(加密资源 + 平台私钥签名)。
func buildWechatNotify(t *testing.T, provider WechatProvider, platformPrivPEM string, plaintext string) Notify {
	t.Helper()
	block, err := aes.NewCipher([]byte(testAPIv3Key))
	if err != nil {
		t.Fatalf("aes cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}
	nonce := "0123456789ab" // 12 字节
	aad := "transaction"
	sealed := gcm.Seal(nil, []byte(nonce), []byte(plaintext), []byte(aad))
	ciphertext := base64.StdEncoding.EncodeToString(sealed)
	body := `{"id":"evt1","event_type":"TRANSACTION.SUCCESS","resource":{"algorithm":"AEAD_AES_256_GCM","ciphertext":"` +
		ciphertext + `","nonce":"` + nonce + `","associated_data":"` + aad + `"}}`

	priv, err := parseRSAPrivateKey(platformPrivPEM)
	if err != nil {
		t.Fatalf("parse platform priv: %v", err)
	}
	timestamp := "1700000000"
	headerNonce := "headernonce"
	signature, err := signRSA2(priv, timestamp+"\n"+headerNonce+"\n"+body+"\n")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	header := http.Header{}
	header.Set("Wechatpay-Timestamp", timestamp)
	header.Set("Wechatpay-Nonce", headerNonce)
	header.Set("Wechatpay-Signature", signature)
	header.Set("Wechatpay-Serial", "PLATFORM_SERIAL")
	return Notify{Header: header, Body: []byte(body)}
}

func TestWechatVerifyNotifyRoundTrip(t *testing.T) {
	platformPriv, platformPub := mustAlipayKeys(t)
	merchantPriv, _ := mustAlipayKeys(t)
	provider, err := NewWechatProvider(WechatConfig{
		AppID:          "wxappid",
		MchID:          "1900000001",
		SerialNo:       "SERIAL123",
		APIv3Key:       testAPIv3Key,
		PrivateKey:     merchantPriv,
		PlatformPublic: platformPub,
	})
	if err != nil {
		t.Fatalf("new wechat provider: %v", err)
	}
	notify := buildWechatNotify(t, provider, platformPriv,
		`{"out_trade_no":"PO20260531","transaction_id":"4200001234","trade_state":"SUCCESS"}`)

	res, err := provider.VerifyNotify(context.Background(), notify)
	if err != nil {
		t.Fatalf("verify notify: %v", err)
	}
	if res.PayNo != "PO20260531" || res.Status != "paid" || res.TransactionID != "4200001234" {
		t.Fatalf("unexpected notify result: %#v", res)
	}
}

func TestWechatVerifyNotifyRejectsTamperedSignature(t *testing.T) {
	platformPriv, platformPub := mustAlipayKeys(t)
	merchantPriv, _ := mustAlipayKeys(t)
	provider, err := NewWechatProvider(WechatConfig{
		AppID:          "wxappid",
		MchID:          "1900000001",
		SerialNo:       "SERIAL123",
		APIv3Key:       testAPIv3Key,
		PrivateKey:     merchantPriv,
		PlatformPublic: platformPub,
	})
	if err != nil {
		t.Fatalf("new wechat provider: %v", err)
	}
	notify := buildWechatNotify(t, provider, platformPriv,
		`{"out_trade_no":"PO1","trade_state":"SUCCESS"}`)
	notify.Header.Set("Wechatpay-Signature", base64.StdEncoding.EncodeToString([]byte("tampered")))
	if _, err := provider.VerifyNotify(context.Background(), notify); err != ErrInvalidSignature {
		t.Fatalf("expected signature error, got %v", err)
	}
}

func TestWechatVerifyNotifyMissingHeaders(t *testing.T) {
	provider := newTestWechatProvider(t, "", nil)
	if _, err := provider.VerifyNotify(context.Background(), Notify{Body: []byte("{}")}); err != ErrInvalidSignature {
		t.Fatalf("expected signature error for missing headers, got %v", err)
	}
}

func TestMapWechatTradeState(t *testing.T) {
	cases := map[string]string{
		"SUCCESS":    "paid",
		"CLOSED":     "failed",
		"REVOKED":    "failed",
		"PAYERROR":   "failed",
		"NOTPAY":     "",
		"USERPAYING": "",
		"":           "",
	}
	for input, want := range cases {
		if got := mapWechatTradeState(input); got != want {
			t.Fatalf("mapWechatTradeState(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNewWechatProviderValidation(t *testing.T) {
	privPEM, pubPEM := mustAlipayKeys(t)
	base := WechatConfig{
		AppID:          "wxappid",
		MchID:          "1900000001",
		SerialNo:       "SERIAL123",
		APIv3Key:       testAPIv3Key,
		PrivateKey:     privPEM,
		PlatformPublic: pubPEM,
	}
	missing := base
	missing.MchID = ""
	if _, err := NewWechatProvider(missing); err == nil {
		t.Fatal("expected error for missing mch_id")
	}
	badKey := base
	badKey.APIv3Key = "tooshort"
	if _, err := NewWechatProvider(badKey); err == nil {
		t.Fatal("expected error for invalid api_v3_key length")
	}
	badPriv := base
	badPriv.PrivateKey = "not-a-key"
	if _, err := NewWechatProvider(badPriv); err == nil {
		t.Fatal("expected error for invalid private key")
	}
}

func TestBuildRegistryWithWechat(t *testing.T) {
	privPEM, pubPEM := mustAlipayKeys(t)
	reg, err := BuildRegistry(RegistryConfig{
		Channels: "mock,wechat",
		Wechat: WechatConfig{
			AppID:          "wxappid",
			MchID:          "1900000001",
			SerialNo:       "SERIAL123",
			APIv3Key:       testAPIv3Key,
			PrivateKey:     privPEM,
			PlatformPublic: pubPEM,
		},
	})
	if err != nil {
		t.Fatalf("build registry: %v", err)
	}
	if !reg.IsEnabled("wechat") || !reg.IsEnabled("mock") {
		t.Fatalf("expected mock and wechat enabled: %#v", reg.Channels())
	}
}

func TestBuildRegistryWechatMissingConfig(t *testing.T) {
	if _, err := BuildRegistry(RegistryConfig{Channels: "wechat"}); err == nil {
		t.Fatal("expected error for missing wechat config")
	}
}
