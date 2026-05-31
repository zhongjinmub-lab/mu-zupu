package payment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 编译期断言:真实渠道实现 OrderQuerier,mock 不实现。
var (
	_ OrderQuerier = AlipayProvider{}
	_ OrderQuerier = WechatProvider{}
)

func TestMockProviderIsNotOrderQuerier(t *testing.T) {
	if _, ok := any(NewMockProvider("")).(OrderQuerier); ok {
		t.Fatal("mock provider should not implement OrderQuerier")
	}
}

func TestAlipayQueryOrderPaid(t *testing.T) {
	appPriv, _ := mustAlipayKeys(t)
	alipayPriv, alipayPub := mustAlipayKeys(t)
	signer, err := parseRSAPrivateKey(alipayPriv)
	if err != nil {
		t.Fatalf("parse alipay priv: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.PostForm.Get("method") != "alipay.trade.query" {
			t.Errorf("unexpected method: %s", r.PostForm.Get("method"))
		}
		node := `{"code":"10000","msg":"Success","trade_no":"2026053122001","out_trade_no":"PO1","trade_status":"TRADE_SUCCESS"}`
		sign, serr := signRSA2(signer, node)
		if serr != nil {
			t.Errorf("sign: %v", serr)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"alipay_trade_query_response":` + node + `,"sign":"` + sign + `"}`))
	}))
	defer server.Close()

	provider, err := NewAlipayProvider(AlipayConfig{
		AppID:      "2021000000000000",
		PrivateKey: appPriv,
		PublicKey:  alipayPub,
		Gateway:    server.URL,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("new alipay provider: %v", err)
	}
	res, err := provider.QueryOrder(context.Background(), "PO1")
	if err != nil {
		t.Fatalf("query order: %v", err)
	}
	if res.Status != QueryStatusPaid || res.TransactionID != "2026053122001" || res.PayNo != "PO1" {
		t.Fatalf("unexpected query result: %#v", res)
	}
}

func TestAlipayQueryOrderNotFound(t *testing.T) {
	appPriv, _ := mustAlipayKeys(t)
	alipayPriv, alipayPub := mustAlipayKeys(t)
	signer, _ := parseRSAPrivateKey(alipayPriv)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		node := `{"code":"40004","msg":"Business Failed","sub_code":"ACQ.TRADE_NOT_EXIST","out_trade_no":"PO404"}`
		sign, _ := signRSA2(signer, node)
		_, _ = w.Write([]byte(`{"alipay_trade_query_response":` + node + `,"sign":"` + sign + `"}`))
	}))
	defer server.Close()

	provider, _ := NewAlipayProvider(AlipayConfig{
		AppID:      "app",
		PrivateKey: appPriv,
		PublicKey:  alipayPub,
		Gateway:    server.URL,
		HTTPClient: server.Client(),
	})
	res, err := provider.QueryOrder(context.Background(), "PO404")
	if err != nil {
		t.Fatalf("query order: %v", err)
	}
	if res.Status != QueryStatusNotFound {
		t.Fatalf("expected not_found, got %#v", res)
	}
}

func TestAlipayQueryOrderRejectsBadSignature(t *testing.T) {
	appPriv, _ := mustAlipayKeys(t)
	_, alipayPub := mustAlipayKeys(t)
	otherPriv, _ := mustAlipayKeys(t)
	wrongSigner, _ := parseRSAPrivateKey(otherPriv) // 与 provider 配置的公钥不匹配

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		node := `{"code":"10000","trade_status":"TRADE_SUCCESS","out_trade_no":"PO1","trade_no":"x"}`
		sign, _ := signRSA2(wrongSigner, node)
		_, _ = w.Write([]byte(`{"alipay_trade_query_response":` + node + `,"sign":"` + sign + `"}`))
	}))
	defer server.Close()

	provider, _ := NewAlipayProvider(AlipayConfig{
		AppID:      "app",
		PrivateKey: appPriv,
		PublicKey:  alipayPub,
		Gateway:    server.URL,
		HTTPClient: server.Client(),
	})
	if _, err := provider.QueryOrder(context.Background(), "PO1"); err != ErrInvalidSignature {
		t.Fatalf("expected signature error, got %v", err)
	}
}

func TestWechatQueryOrderPaid(t *testing.T) {
	merchantPriv, _ := mustAlipayKeys(t)
	platformPriv, platformPub := mustAlipayKeys(t)
	signer, err := parseRSAPrivateKey(platformPriv)
	if err != nil {
		t.Fatalf("parse platform priv: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("missing authorization header")
		}
		body := `{"out_trade_no":"PO1","transaction_id":"4200001234","trade_state":"SUCCESS"}`
		ts := "1700000000"
		nonce := "respnonce"
		sign, _ := signRSA2(signer, ts+"\n"+nonce+"\n"+body+"\n")
		w.Header().Set("Wechatpay-Timestamp", ts)
		w.Header().Set("Wechatpay-Nonce", nonce)
		w.Header().Set("Wechatpay-Signature", sign)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	provider, err := NewWechatProvider(WechatConfig{
		AppID:          "wxappid",
		MchID:          "1900000001",
		SerialNo:       "SERIAL123",
		APIv3Key:       testAPIv3Key,
		PrivateKey:     merchantPriv,
		PlatformPublic: platformPub,
		Gateway:        server.URL,
		HTTPClient:     server.Client(),
	})
	if err != nil {
		t.Fatalf("new wechat provider: %v", err)
	}
	res, err := provider.QueryOrder(context.Background(), "PO1")
	if err != nil {
		t.Fatalf("query order: %v", err)
	}
	if res.Status != QueryStatusPaid || res.TransactionID != "4200001234" {
		t.Fatalf("unexpected query result: %#v", res)
	}
}

func TestWechatQueryOrderNotFound(t *testing.T) {
	merchantPriv, _ := mustAlipayKeys(t)
	_, platformPub := mustAlipayKeys(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"code":"ORDER_NOT_EXIST","message":"订单不存在"}`))
	}))
	defer server.Close()

	provider, _ := NewWechatProvider(WechatConfig{
		AppID:          "wxappid",
		MchID:          "1900000001",
		SerialNo:       "SERIAL123",
		APIv3Key:       testAPIv3Key,
		PrivateKey:     merchantPriv,
		PlatformPublic: platformPub,
		Gateway:        server.URL,
		HTTPClient:     server.Client(),
	})
	res, err := provider.QueryOrder(context.Background(), "PO404")
	if err != nil {
		t.Fatalf("query order: %v", err)
	}
	if res.Status != QueryStatusNotFound {
		t.Fatalf("expected not_found, got %#v", res)
	}
}

func TestExtractJSONObjectNode(t *testing.T) {
	body := []byte(`{"alipay_trade_query_response":{"a":"1","nested":{"b":"2"},"s":"has}brace{inside"},"sign":"xx"}`)
	node, ok := extractJSONObjectNode(body, "alipay_trade_query_response")
	if !ok {
		t.Fatal("expected node extracted")
	}
	if node != `{"a":"1","nested":{"b":"2"},"s":"has}brace{inside"}` {
		t.Fatalf("unexpected node: %s", node)
	}
}
