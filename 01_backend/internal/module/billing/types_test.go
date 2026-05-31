package billing

import "testing"

func TestRecordUsageInputNormalizeAndValidate(t *testing.T) {
	in := RecordUsageInput{
		TenantID:    " tenant ",
		SubjectType: " agent ",
		SubjectID:   " subject ",
		Metric:      " agent_messages ",
		Quantity:    1,
		RequestID:   " req ",
	}
	in.Normalize()
	if in.TenantID != "tenant" || in.SubjectType != "agent" || in.SubjectID != "subject" || in.Metric != MetricAgentMessages || in.Unit != "count" || in.RequestID != "req" {
		t.Fatalf("normalized input = %#v", in)
	}
	if err := in.Validate(); err != nil {
		t.Fatalf("expected valid usage input: %v", err)
	}
}

func TestRecordUsageInputValidateRequiresPositiveQuantity(t *testing.T) {
	in := RecordUsageInput{TenantID: "tenant", SubjectType: "agent", Metric: MetricAgentMessages}
	if err := in.Validate(); err == nil {
		t.Fatal("expected quantity validation error")
	}
}

func TestQuotaLimitReadsNumericQuota(t *testing.T) {
	limit, ok := quotaLimit(map[string]any{MetricRAGRequests: float64(1000)}, MetricRAGRequests)
	if !ok || limit != 1000 {
		t.Fatalf("limit = %v, ok = %v", limit, ok)
	}
}

func TestQuotaCheckError(t *testing.T) {
	err := QuotaCheck{Metric: MetricAgentMessages, Used: 10, Requested: 1, Limit: 10}
	if err.Error() == "" {
		t.Fatal("expected quota error message")
	}
}

func TestQuotaCheckLimitedFlag(t *testing.T) {
	check := QuotaCheck{Metric: MetricRAGRequests, Limit: 100, Used: 40, Remaining: 60, Allowed: true, Limited: true}
	if !check.Limited || !check.Allowed || check.Remaining != 60 {
		t.Fatalf("quota check = %#v", check)
	}
}

func TestCreateOrderRequestNormalizeAndValidate(t *testing.T) {
	req := CreateOrderRequest{PlanCode: " free ", Currency: "", Metadata: nil}
	req.Normalize()
	if req.OrderType != "subscription" || req.PlanCode != "free" || req.Currency != "CNY" || req.Metadata == nil {
		t.Fatalf("normalized order request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid order request: %v", err)
	}
}

func TestCreatePaymentOrderRequestNormalizeDefaultsChannel(t *testing.T) {
	req := CreatePaymentOrderRequest{BusinessOrderID: " order "}
	req.Normalize()
	if req.BusinessOrderID != "order" || req.Channel != "mock" {
		t.Fatalf("normalized request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid request: %v", err)
	}
}

func TestCreatePaymentOrderRequestAcceptsRealChannel(t *testing.T) {
	req := CreatePaymentOrderRequest{BusinessOrderID: "order", Channel: "Alipay"}
	req.Normalize()
	if req.Channel != "alipay" {
		t.Fatalf("channel should be lowercased, got %q", req.Channel)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected real channel to be accepted: %v", err)
	}
}

func TestCreatePaymentOrderRequestRequiresBusinessOrder(t *testing.T) {
	req := CreatePaymentOrderRequest{Channel: "alipay"}
	req.Normalize()
	if err := req.Validate(); err == nil {
		t.Fatal("expected business_order_id validation error")
	}
}

func TestPaymentCallbackRequestNormalizeAndValidate(t *testing.T) {
	req := PaymentCallbackRequest{PayNo: " pay ", TransactionID: " tx "}
	req.Normalize()
	if req.PayNo != "pay" || req.TransactionID != "tx" || req.Status != "paid" || req.Metadata == nil {
		t.Fatalf("normalized callback request = %#v", req)
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected valid callback request: %v", err)
	}
}

func TestVerifyPaymentCallbackSignatureAllowsEmptySecret(t *testing.T) {
	if !verifyPaymentCallbackSignature("", []byte(`{"pay_no":"pay"}`), "") {
		t.Fatal("empty secret should allow callback without signature")
	}
}

func TestVerifyPaymentCallbackSignatureAcceptsValidSignature(t *testing.T) {
	body := []byte(`{"pay_no":"pay","status":"paid"}`)
	signature := paymentCallbackSignature("secret", body)
	if !verifyPaymentCallbackSignature("secret", body, signature) {
		t.Fatal("expected valid payment callback signature")
	}
}

func TestVerifyPaymentCallbackSignatureRejectsInvalidSignature(t *testing.T) {
	body := []byte(`{"pay_no":"pay","status":"paid"}`)
	if verifyPaymentCallbackSignature("secret", body, "sha256=bad") {
		t.Fatal("expected invalid payment callback signature to be rejected")
	}
	if verifyPaymentCallbackSignature("secret", body, "") {
		t.Fatal("expected missing payment callback signature to be rejected")
	}
}

func TestNormalizeReasonTrimsAndLimitsLength(t *testing.T) {
	long := " " + string(make([]byte, 600)) + " "
	got := normalizeReason(long)
	if len(got) != 512 {
		t.Fatalf("reason length = %d", len(got))
	}
}
