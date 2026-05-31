import request from './request'

// 渠道类型目录
export function listChannelTypes() {
  return request.get('/channel-types')
}

// 渠道列表
export function listChannels() {
  return request.get('/channels')
}

// 渠道概览统计
export function channelsSummary() {
  return request.get('/channels/summary')
}

// 渠道详情
export function getChannel(channelId: string) {
  return request.get(`/channels/${channelId}`)
}

// 渠道接入代码与说明
export function getChannelEmbed(channelId: string) {
  return request.get(`/channels/${channelId}/embed`)
}

// 创建渠道
export function createChannel(data: {
  agent_id: string
  type: string
  name: string
  config?: Record<string, any>
}) {
  return request.post('/channels', data)
}

// 更新渠道
export function updateChannel(channelId: string, data: { name?: string; config?: Record<string, any> }) {
  return request.put(`/channels/${channelId}`, data)
}

// 复制渠道
export function duplicateChannel(channelId: string) {
  return request.post(`/channels/${channelId}/duplicate`)
}

// 启用渠道
export function enableChannel(channelId: string) {
  return request.post(`/channels/${channelId}/enable`)
}

// 禁用渠道
export function disableChannel(channelId: string) {
  return request.post(`/channels/${channelId}/disable`)
}

// 归档渠道
export function archiveChannel(channelId: string) {
  return request.delete(`/channels/${channelId}`)
}
