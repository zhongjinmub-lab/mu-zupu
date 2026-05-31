package billing

import (
	"testing"

	"mu-agent-saas/internal/payment"
)

func TestHandlerNotifyURL(t *testing.T) {
	h := Handler{NotifyBaseURL: "https://api.example.com"}
	if got := h.notifyURL("alipay"); got != "https://api.example.com/api/v1/payment-notify/alipay" {
		t.Fatalf("notifyURL = %q", got)
	}
	if got := (Handler{}).notifyURL("alipay"); got != "" {
		t.Fatalf("expected empty notify url without base, got %q", got)
	}
}

func TestPrepaySummaryKeepsNonSensitiveFields(t *testing.T) {
	summary := prepaySummary(payment.Prepay{
		Channel: "alipay",
		Method:  "redirect",
		PayURL:  "https://openapi.alipay.com/gateway.do?sign=abc",
		Message: "请在新页面跳转到支付宝完成支付。",
	})
	if summary["channel"] != "alipay" || summary["method"] != "redirect" {
		t.Fatalf("summary = %#v", summary)
	}
	if summary["pay_url"] != "https://openapi.alipay.com/gateway.do?sign=abc" {
		t.Fatalf("pay_url missing: %#v", summary)
	}
	if summary["message"] == "" {
		t.Fatalf("message missing: %#v", summary)
	}

	mockSummary := prepaySummary(payment.Prepay{Channel: "mock", Method: "mock"})
	if _, ok := mockSummary["pay_url"]; ok {
		t.Fatalf("mock summary should not include pay_url: %#v", mockSummary)
	}
	if _, ok := mockSummary["qr_content"]; ok {
		t.Fatalf("mock summary should not include qr_content: %#v", mockSummary)
	}
}
