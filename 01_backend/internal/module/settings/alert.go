package settings

// AlertRule 表示一条监控告警规则。
type AlertRule struct {
	Metric      string  `json:"metric"`
	Operator    string  `json:"operator"`
	Threshold   float64 `json:"threshold"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
}

// AlertPolicy 表示监控告警策略（内置默认规则集合）。
type AlertPolicy struct {
	Enabled bool        `json:"enabled"`
	Rules   []AlertRule `json:"rules"`
	Notes   []string    `json:"notes"`
}

// AlertTriggered 表示一条被触发的告警。
type AlertTriggered struct {
	Metric      string  `json:"metric"`
	Operator    string  `json:"operator"`
	Threshold   float64 `json:"threshold"`
	Actual      float64 `json:"actual"`
	Severity    string  `json:"severity"`
	Description string  `json:"description"`
}

// AlertEvaluationResult 表示一次告警评估结果。
type AlertEvaluationResult struct {
	Healthy       bool             `json:"healthy"`
	WarningCount  int              `json:"warning_count"`
	CriticalCount int              `json:"critical_count"`
	Triggered     []AlertTriggered `json:"triggered"`
}

// DefaultAlertPolicy 返回内置监控告警默认策略。纯函数。
func DefaultAlertPolicy() AlertPolicy {
	return AlertPolicy{
		Enabled: true,
		Rules: []AlertRule{
			{Metric: "heap_alloc_mb", Operator: ">=", Threshold: 768, Severity: "warning", Description: "堆内存使用偏高（≥768MB）"},
			{Metric: "heap_alloc_mb", Operator: ">=", Threshold: 1536, Severity: "critical", Description: "堆内存使用过高（≥1536MB），存在 OOM 风险"},
			{Metric: "goroutines", Operator: ">=", Threshold: 2000, Severity: "warning", Description: "Goroutine 数量偏高（≥2000），疑似泄漏"},
			{Metric: "goroutines", Operator: ">=", Threshold: 10000, Severity: "critical", Description: "Goroutine 数量过高（≥10000）"},
			{Metric: "error_rate", Operator: ">=", Threshold: 0.05, Severity: "warning", Description: "错误率偏高（≥5%）"},
			{Metric: "error_rate", Operator: ">=", Threshold: 0.2, Severity: "critical", Description: "错误率过高（≥20%）"},
			{Metric: "p95_latency_ms", Operator: ">=", Threshold: 1000, Severity: "warning", Description: "P95 延迟偏高（≥1s）"},
			{Metric: "p95_latency_ms", Operator: ">=", Threshold: 3000, Severity: "critical", Description: "P95 延迟过高（≥3s）"},
		},
		Notes: []string{
			"告警阈值为内置默认值，后续版本可支持租户级自定义。",
			"评估为无副作用的纯函数；接入真实告警通道（邮件/Webhook/IM）为后续增强。",
			"error_rate 取值范围 0~1；p95_latency_ms 与 heap_alloc_mb 为绝对值。",
		},
	}
}

// EvaluateAlerts 按规则评估给定指标，返回触发的告警与健康状态。纯函数。
func EvaluateAlerts(metrics map[string]float64, rules []AlertRule) AlertEvaluationResult {
	result := AlertEvaluationResult{Healthy: true, Triggered: make([]AlertTriggered, 0)}
	for _, rule := range rules {
		actual, ok := metrics[rule.Metric]
		if !ok {
			continue
		}
		if !compareAlertMetric(actual, rule.Operator, rule.Threshold) {
			continue
		}
		result.Triggered = append(result.Triggered, AlertTriggered{
			Metric:      rule.Metric,
			Operator:    rule.Operator,
			Threshold:   rule.Threshold,
			Actual:      actual,
			Severity:    rule.Severity,
			Description: rule.Description,
		})
		if rule.Severity == "critical" {
			result.CriticalCount++
		} else {
			result.WarningCount++
		}
	}
	result.Healthy = len(result.Triggered) == 0
	return result
}

// compareAlertMetric 按运算符比较指标值与阈值。
func compareAlertMetric(value float64, op string, threshold float64) bool {
	switch op {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}
