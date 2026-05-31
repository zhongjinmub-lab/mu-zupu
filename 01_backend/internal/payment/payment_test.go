package payment

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func mustAlipayKeys(t *testing.T) (privPEM, pubPEM string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}))
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return privPEM, pubPEM
}

func TestMockProviderVerifyNotify(t *testing.T) {
	provider := NewMockProvider("secret")
	body := []byte(`{"pay_no":"PO123","status":"paid","transaction_id":"tx-1"}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	header := http.Header{}
	header.Set("X-Payment-Signature", signature)
	res, err := provider.VerifyNotify(context.Background(), Notify{Header: header, Body: body})
	if err != nil {
		t.Fatalf("expected valid notify: %v", err)
	}
	if res.PayNo != "PO123" || res.Status != "paid" || res.TransactionID != "tx-1" {
		t.Fatalf("unexpected notify result: %#v", res)
	}

	if _, err := provider.VerifyNotify(context.Background(), Notify{Body: body}); err != ErrInvalidSignature {
		t.Fatalf("expected signature error without header, got %v", err)
	}

	bad := http.Header{}
	bad.Set("X-Payment-Signature", "sha256=deadbeef")
	if _, err := provider.VerifyNotify(context.Background(), Notify{Header: bad, Body: body}); err != ErrInvalidSignature {
		t.Fatalf("expected signature error for bad signature, got %v", err)
	}
}

func TestMockProviderEmptySecretAllowsNotify(t *testing.T) {
	provider := NewMockProvider("")
	body := []byte(`{"pay_no":"PO9","status":"failed"}`)
	res, err := provider.VerifyNotify(context.Background(), Notify{Body: body})
	if err != nil {
		t.Fatalf("empty secret should allow notify: %v", err)
	}
	if res.PayNo != "PO9" || res.Status != "failed" {
		t.Fatalf("unexpected notify result: %#v", res)
	}
}

func TestMockProviderRejectsEmptyPayNo(t *testing.T) {
	provider := NewMockProvider("")
	if _, err := provider.VerifyNotify(context.Background(), Notify{Body: []byte(`{"status":"paid"}`)}); err != ErrInvalidNotify {
		t.Fatalf("expected invalid notify for empty pay_no, got %v", err)
	}
}

func TestMockProviderCreatePrepay(t *testing.T) {
	provider := NewMockProvider("")
	prepay, err := provider.CreatePrepay(context.Background(), PrepayInput{PayNo: "PO1", AmountCents: 100})
	if err != nil {
		t.Fatalf("create prepay: %v", err)
	}
	if prepay.Channel != "mock" || prepay.Method != "mock" || prepay.Extra["pay_no"] != "PO1" {
		t.Fatalf("unexpected prepay: %#v", prepay)
	}
}

func TestBuildRegistryDefaultsToMock(t *testing.T) {
	reg, err := BuildRegistry(RegistryConfig{})
	if err != nil {
		t.Fatalf("build registry: %v", err)
	}
	if !reg.IsEnabled("mock") {
		t.Fatal("expected mock channel enabled by default")
	}
	if reg.IsEnabled("alipay") {
		t.Fatal("alipay should not be enabled by default")
	}
	if got := reg.Channels(); len(got) != 1 || got[0] != "mock" {
		t.Fatalf("unexpected channels: %#v", got)
	}
}

func TestBuildRegistryUnknownChannel(t *testing.T) {
	if _, err := BuildRegistry(RegistryConfig{Channels: "mock,paypal"}); err == nil {
		t.Fatal("expected error for unknown channel")
	}
}

func TestBuildRegistryAlipayMissingConfig(t *testing.T) {
	if _, err := BuildRegistry(RegistryConfig{Channels: "alipay"}); err == nil {
		t.Fatal("expected error for missing alipay config")
	}
}

func TestBuildRegistryWithAlipay(t *testing.T) {
	privPEM, pubPEM := mustAlipayKeys(t)
	reg, err := BuildRegistry(RegistryConfig{
		Channels: "mock,alipay",
		Alipay:   AlipayConfig{AppID: "2021000000000000", PrivateKey: privPEM, PublicKey: pubPEM},
	})
	if err != nil {
		t.Fatalf("build registry: %v", err)
	}
	if !reg.IsEnabled("mock") || !reg.IsEnabled("alipay") {
		t.Fatalf("expected mock and alipay enabled: %#v", reg.Channels())
	}
	if _, ok := reg.Get("ALIPAY"); !ok {
		t.Fatal("channel lookup should be case-insensitive")
	}
}

func TestAlipayCreatePrepayBuildsSignedURL(t *testing.T) {
	privPEM, pubPEM := mustAlipayKeys(t)
	provider, err := NewAlipayProvider(AlipayConfig{AppID: "2021000000000000", PrivateKey: privPEM, PublicKey: pubPEM})
	if err != nil {
		t.Fatalf("new alipay provider: %v", err)
	}
	prepay, err := provider.CreatePrepay(context.Background(), PrepayInput{
		PayNo:       "PO20260531",
		AmountCents: 12345,
		Currency:    "CNY",
		Subject:     "套餐订阅",
		NotifyURL:   "https://api.example.com/api/v1/payments/notify/alipay",
		ReturnURL:   "https://app.example.com/return",
	})
	if err != nil {
		t.Fatalf("create prepay: %v", err)
	}
	if prepay.Method != "redirect" || prepay.Channel != "alipay" {
		t.Fatalf("unexpected prepay: %#v", prepay)
	}
	u, err := url.Parse(prepay.PayURL)
	if err != nil {
		t.Fatalf("parse pay url: %v", err)
	}
	q := u.Query()
	if q.Get("method") != "alipay.trade.page.pay" {
		t.Fatalf("unexpected method: %s", q.Get("method"))
	}
	if q.Get("sign") == "" {
		t.Fatal("expected sign in pay url")
	}
	if !strings.Contains(q.Get("biz_content"), "PO20260531") {
		t.Fatalf("biz_content missing out_trade_no: %s", q.Get("biz_content"))
	}
	if !strings.Contains(q.Get("biz_content"), "123.45") {
		t.Fatalf("biz_content missing total_amount: %s", q.Get("biz_content"))
	}
}

func TestAlipayVerifyNotifyRoundTrip(t *testing.T) {
	privPEM, pubPEM := mustAlipayKeys(t)
	provider, err := NewAlipayProvider(AlipayConfig{AppID: "2021000000000000", PrivateKey: privPEM, PublicKey: pubPEM})
	if err != nil {
		t.Fatalf("new alipay provider: %v", err)
	}
	params := map[string]string{
		"app_id":       "2021000000000000",
		"out_trade_no": "PO20260531",
		"trade_no":     "2026053122001",
		"trade_status": "TRADE_SUCCESS",
		"total_amount": "123.45",
		"sign_type":    "RSA2",
	}
	sign, err := signRSA2(provider.priv, buildSignContent(params, true))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	form.Set("sign", sign)

	res, err := provider.VerifyNotify(context.Background(), Notify{Body: []byte(form.Encode())})
	if err != nil {
		t.Fatalf("verify notify: %v", err)
	}
	if res.PayNo != "PO20260531" || res.Status != "paid" || res.TransactionID != "2026053122001" {
		t.Fatalf("unexpected notify result: %#v", res)
	}
	if _, ok := res.Raw["sign"]; ok {
		t.Fatal("raw payload should not retain sign")
	}
}

func TestAlipayVerifyNotifyRejectsTamper(t *testing.T) {
	privPEM, pubPEM := mustAlipayKeys(t)
	provider, err := NewAlipayProvider(AlipayConfig{AppID: "2021000000000000", PrivateKey: privPEM, PublicKey: pubPEM})
	if err != nil {
		t.Fatalf("new alipay provider: %v", err)
	}
	params := map[string]string{
		"out_trade_no": "PO1",
		"trade_status": "TRADE_SUCCESS",
		"sign_type":    "RSA2",
	}
	sign, err := signRSA2(provider.priv, buildSignContent(params, true))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	form.Set("sign", sign)
	// 篡改金额后验签必须失败。
	form.Set("total_amount", "0.01")
	if _, err := provider.VerifyNotify(context.Background(), Notify{Body: []byte(form.Encode())}); err != ErrInvalidSignature {
		t.Fatalf("expected signature error after tamper, got %v", err)
	}
}

func TestAlipayVerifyNotifyIgnoresPendingStatus(t *testing.T) {
	privPEM, pubPEM := mustAlipayKeys(t)
	provider, err := NewAlipayProvider(AlipayConfig{AppID: "app", PrivateKey: privPEM, PublicKey: pubPEM})
	if err != nil {
		t.Fatalf("new alipay provider: %v", err)
	}
	params := map[string]string{
		"out_trade_no": "PO1",
		"trade_status": "WAIT_BUYER_PAY",
		"sign_type":    "RSA2",
	}
	sign, err := signRSA2(provider.priv, buildSignContent(params, true))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	form.Set("sign", sign)
	if _, err := provider.VerifyNotify(context.Background(), Notify{Body: []byte(form.Encode())}); err == nil {
		t.Fatal("expected error for non-final trade status")
	}
}

func TestNewAlipayProviderValidation(t *testing.T) {
	privPEM, pubPEM := mustAlipayKeys(t)
	if _, err := NewAlipayProvider(AlipayConfig{PrivateKey: privPEM, PublicKey: pubPEM}); err == nil {
		t.Fatal("expected error for missing app_id")
	}
	if _, err := NewAlipayProvider(AlipayConfig{AppID: "app", PrivateKey: "bad", PublicKey: pubPEM}); err == nil {
		t.Fatal("expected error for invalid private key")
	}
	if _, err := NewAlipayProvider(AlipayConfig{AppID: "app", PrivateKey: privPEM, PublicKey: pubPEM, SignType: "RSA"}); err == nil {
		t.Fatal("expected error for unsupported sign type")
	}
}

func TestMapAlipayTradeStatus(t *testing.T) {
	cases := map[string]string{
		"TRADE_SUCCESS":  "paid",
		"TRADE_FINISHED": "paid",
		"TRADE_CLOSED":   "failed",
		"WAIT_BUYER_PAY": "",
		"":               "",
	}
	for input, want := range cases {
		if got := mapAlipayTradeStatus(input); got != want {
			t.Fatalf("mapAlipayTradeStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFormatYuan(t *testing.T) {
	cases := map[int64]string{
		0:     "0.00",
		5:     "0.05",
		99:    "0.99",
		100:   "1.00",
		12345: "123.45",
	}
	for input, want := range cases {
		if got := formatYuan(input); got != want {
			t.Fatalf("formatYuan(%d) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildSignContentSortsAndFilters(t *testing.T) {
	params := map[string]string{
		"b":         "2",
		"a":         "1",
		"empty":     "",
		"sign":      "should-skip",
		"sign_type": "RSA2",
	}
	if got := buildSignContent(params, true); got != "a=1&b=2" {
		t.Fatalf("excludeSignType content = %q", got)
	}
	if got := buildSignContent(params, false); got != "a=1&b=2&sign_type=RSA2" {
		t.Fatalf("keep sign_type content = %q", got)
	}
}
