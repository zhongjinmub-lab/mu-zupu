import request from './request'

// 智能体列表
export function listAgents() {
  return request.get('/agents')
}

// 获取智能体详情
export function getAgent(agentId: string) {
  return request.get(`/agents/${agentId}`)
}

// 创建智能体
export function createAgent(data: {
  name: string
  code: string
  description?: string
  system_prompt?: string
}) {
  return request.post('/agents', data)
}

// 更新智能体
export function updateAgent(agentId: string, data: Record<string, any>) {
  return request.put(`/agents/${agentId}`, data)
}

// 发布智能体
export function publishAgent(agentId: string) {
  return request.post(`/agents/${agentId}/publish`)
}

// 归档智能体
export function archiveAgent(agentId: string) {
  return request.delete(`/agents/${agentId}`)
}

// ========== 版本管理 ==========

// 列出版本
export function listVersions(agentId: string, limit = 50) {
  return request.get(`/agents/${agentId}/versions`, { params: { limit } })
}

// 获取版本详情
export function getVersion(agentId: string, versionId: string) {
  return request.get(`/agents/${agentId}/versions/${versionId}`)
}

// 创建版本
export function createVersion(agentId: string, data: {
  version_no: string
  prompt?: string
  channel?: string
  publish_note?: string
  model_config?: Record<string, any>
  tool_config?: Record<string, any>
  knowledge_config?: Record<string, any>
}) {
  return request.post(`/agents/${agentId}/versions`, data)
}

// 发布版本
export function publishVersion(agentId: string, versionId: string, publishNote?: string) {
  return request.post(`/agents/${agentId}/versions/${versionId}/publish`, { publish_note: publishNote })
}

// 回滚版本
export function rollbackVersion(agentId: string, versionId: string) {
  return request.post(`/agents/${agentId}/versions/${versionId}/rollback`)
}

// ========== 族谱 ==========

// 获取族谱图
export function getGenealogyGraph(params?: { q?: string; relation_type?: string }) {
  return request.get('/agent-genealogy/graph', { params })
}

// 创建族谱边
export function createGenealogyEdge(data: {
  parent_agent_id?: string
  child_agent_id: string
  relation_type?: string
}) {
  return request.post('/agent-genealogy/edges', data)
}
