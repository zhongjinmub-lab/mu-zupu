import request from './request'

// 知识库列表
export function listKnowledgeBases() {
  return request.get('/kbs')
}

// 创建知识库
export function createKnowledgeBase(data: {
  name: string
  code: string
  embedding_model?: string
  embedding_dim?: number
}) {
  return request.post('/kbs', data)
}

// 文档列表
export function listDocuments(kbId: string) {
  return request.get(`/kbs/${kbId}/documents`)
}

// 文档详情
export function getDocument(kbId: string, docId: string) {
  return request.get(`/kbs/${kbId}/documents/${docId}`)
}

// 归档文档
export function archiveDocument(kbId: string, docId: string) {
  return request.delete(`/kbs/${kbId}/documents/${docId}`)
}

// Chunk 列表
export function listDocumentChunks(kbId: string, docId: string) {
  return request.get(`/kbs/${kbId}/documents/${docId}/chunks`)
}

// 从文件创建文档
export function createDocumentFromFile(kbId: string, data: { file_id: string; title: string }) {
  return request.post(`/kbs/${kbId}/documents/from-file`, data)
}

// 知识库问答
export function askKnowledgeBase(kbId: string, data: { query: string; top_k?: number }) {
  return request.post(`/kbs/${kbId}/ask`, data)
}
