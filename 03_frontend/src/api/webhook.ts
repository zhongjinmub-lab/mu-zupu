import request from './request'

// Webhook 端点列表
export function listWebhooks() {
  return request.get('/webhooks')
}

// 创建 Webhook 端点
export function createWebhook(data: {
  url: string
  events: string[]
  secret?: string
  description?: string
}) {
  return request.post('/webhooks', data)
}

// 更新 Webhook 端点
export function updateWebhook(webhookId: string, data: Record<string, any>) {
  return request.put(`/webhooks/${webhookId}`, data)
}

// 删除 Webhook 端点
export function deleteWebhook(webhookId: string) {
  return request.delete(`/webhooks/${webhookId}`)
}

// 测试 Webhook
export function testWebhook(webhookId: string) {
  return request.post(`/webhooks/${webhookId}/test`)
}

// 投递记录列表
export function listDeliveries(params?: { webhook_id?: string; status?: string; limit?: number }) {
  return request.get('/webhook-deliveries', { params })
}

// 投递摘要
export function deliverySummary() {
  return request.get('/webhook-deliveries/summary')
}

// 重试投递
export function retryDelivery(deliveryId: string) {
  return request.post(`/webhook-deliveries/${deliveryId}/retry`)
}
