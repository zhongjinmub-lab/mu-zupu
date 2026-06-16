import request from './request'

// 限流策略
export function getRateLimitPolicy() {
  return request.get('/settings/rate-limit')
}

// 运行配置
export function getRuntimeSummary() {
  return request.get('/settings/runtime')
}

// 监控快照
export function getMonitoringSnapshot() {
  return request.get('/settings/monitoring')
}

// 敏感字段保护
export function getSensitiveFieldSummary() {
  return request.get('/settings/sensitive-fields')
}

// 限流与审计闭环摘要
export function getRateLimitAuditSummary() {
  return request.get('/settings/rate-limit-audit')
}

// 向量检索健康
export function getVectorSearchSummary() {
  return request.get('/settings/vector-search')
}

// 告警策略
export function getAlertPolicy() {
  return request.get('/settings/alert-policy')
}

// 告警状态
export function getAlertStatus() {
  return request.get('/settings/alert-status')
}
