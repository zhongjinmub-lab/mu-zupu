package settings

import "testing"

func TestDefaultAlertPolicyHasRules(t *testing.T) {
	policy := DefaultAlertPolicy()
	if !policy.Enabled || len(policy.Rules) == 0 {
		t.Fatalf("default alert policy should be enabled with rules: %#v", policy)
	}
	for _, r := range policy.Rules {
		if r.Metric == "" || r.Operator == "" || (r.Severity != "warning" && r.Severity != "critical") {
			t.Fatalf("invalid rule: %#v", r)
		}
	}
}

func TestEvaluateAlertsTriggers(t *testing.T) {
	rules := DefaultAlertPolicy().Rules
	// 堆内存 1600MB 应同时触发 warning(>=768) 与 critical(>=1536)
	result := EvaluateAlerts(map[string]float64{"heap_alloc_mb": 1600}, rules)
	if result.Healthy {
		t.Fatal("expected unhealthy when heap is high")
	}
	if result.CriticalCount < 1 || result.WarningCount < 1 {
		t.Fatalf("expected both warning and critical, got %#v", result)
	}
}

func TestEvaluateAlertsHealthy(t *testing.T) {
	rules := DefaultAlertPolicy().Rules
	result := EvaluateAlerts(map[string]float64{"heap_alloc_mb": 100, "goroutines": 50}, rules)
	if !result.Healthy || len(result.Triggered) != 0 {
		t.Fatalf("expected healthy with low metrics, got %#v", result)
	}
}

func TestEvaluateAlertsIgnoresUnknownMetric(t *testing.T) {
	rules := []AlertRule{{Metric: "absent", Operator: ">=", Threshold: 1, Severity: "warning", Description: "x"}}
	result := EvaluateAlerts(map[string]float64{"other": 999}, rules)
	if !result.Healthy {
		t.Fatalf("missing metric should not trigger: %#v", result)
	}
}

func TestCompareAlertMetric(t *testing.T) {
	cases := []struct {
		v   float64
		op  string
		th  float64
		exp bool
	}{
		{10, ">=", 10, true},
		{9, ">=", 10, false},
		{11, ">", 10, true},
		{5, "<", 10, true},
		{10, "==", 10, true},
		{10, "!=", 10, false},
		{1, "??", 1, false},
	}
	for _, tc := range cases {
		if got := compareAlertMetric(tc.v, tc.op, tc.th); got != tc.exp {
			t.Fatalf("compare(%v,%q,%v)=%v want %v", tc.v, tc.op, tc.th, got, tc.exp)
		}
	}
}
