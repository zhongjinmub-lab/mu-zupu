<template>
  <div class="knowledge-view">
    <div class="page-header">
      <h2>知识库管理</h2>
      <el-button type="primary" @click="showCreateDialog = true">新建知识库</el-button>
    </div>

    <!-- 知识库列表 -->
    <el-table :data="knowledgeBases" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="code" label="编码" min-width="120" />
      <el-table-column prop="embedding_model" label="Embedding 模型" min-width="180" />
      <el-table-column prop="embedding_dim" label="向量维度" width="100" />
      <el-table-column prop="status" label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
            {{ row.status }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="170">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="viewDocuments(row)">文档</el-button>
          <el-button link type="primary" @click="openAsk(row)">问答测试</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 文档列表抽屉 -->
    <el-drawer v-model="showDocDrawer" :title="`文档列表 - ${currentKB?.name || ''}`" size="60%">
      <el-table :data="documents" v-loading="docLoading" stripe size="small">
        <el-table-column prop="title" label="标题" min-width="160" />
        <el-table-column prop="source_type" label="来源" width="80" />
        <el-table-column prop="parse_status" label="解析" width="80">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.parse_status)" size="small">{{ row.parse_status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="embedding_status" label="向量化" width="80">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.embedding_status)" size="small">{{ row.embedding_status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="160">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-popconfirm title="确认归档此文档？" @confirm="handleArchiveDoc(row.id)">
              <template #reference>
                <el-button link type="danger" size="small">归档</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-drawer>

    <!-- RAG 问答测试对话框 -->
    <el-dialog v-model="showAskDialog" :title="`RAG 问答 - ${currentKB?.name || ''}`" width="600px">
      <el-form @submit.prevent="handleAsk">
        <el-form-item label="问题">
          <el-input v-model="askQuery" type="textarea" :rows="3" placeholder="输入查询问题..." />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="asking" @click="handleAsk">发送</el-button>
        </el-form-item>
      </el-form>
      <div v-if="askAnswer" class="ask-result">
        <el-divider>回答</el-divider>
        <div class="answer-text">{{ askAnswer }}</div>
      </div>
    </el-dialog>

    <!-- 创建知识库对话框 -->
    <el-dialog v-model="showCreateDialog" title="新建知识库" width="480px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="名称" required>
          <el-input v-model="createForm.name" placeholder="知识库名称" />
        </el-form-item>
        <el-form-item label="编码" required>
          <el-input v-model="createForm.code" placeholder="唯一编码" />
        </el-form-item>
        <el-form-item label="Embedding">
          <el-input v-model="createForm.embedding_model" placeholder="text-embedding-3-small" />
        </el-form-item>
        <el-form-item label="向量维度">
          <el-select v-model="createForm.embedding_dim">
            <el-option :value="384" label="384" />
            <el-option :value="512" label="512" />
            <el-option :value="768" label="768" />
            <el-option :value="1024" label="1024" />
            <el-option :value="1536" label="1536" />
            <el-option :value="3072" label="3072" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  listKnowledgeBases, createKnowledgeBase,
  listDocuments as fetchDocs, archiveDocument, askKnowledgeBase
} from '@/api/kb'

const knowledgeBases = ref<any[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const creating = ref(false)

const createForm = ref({
  name: '',
  code: '',
  embedding_model: 'text-embedding-3-small',
  embedding_dim: 1536,
})

// 文档相关
const showDocDrawer = ref(false)
const docLoading = ref(false)
const documents = ref<any[]>([])
const currentKB = ref<any>(null)

// 问答相关
const showAskDialog = ref(false)
const askQuery = ref('')
const askAnswer = ref('')
const asking = ref(false)

onMounted(() => fetchList())

async function fetchList() {
  loading.value = true
  try {
    const res = await listKnowledgeBases()
    knowledgeBases.value = res.data.data.items || []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!createForm.value.name || !createForm.value.code) {
    ElMessage.warning('请填写名称和编码')
    return
  }
  creating.value = true
  try {
    await createKnowledgeBase(createForm.value)
    ElMessage.success('知识库创建成功')
    showCreateDialog.value = false
    createForm.value = { name: '', code: '', embedding_model: 'text-embedding-3-small', embedding_dim: 1536 }
    await fetchList()
  } finally {
    creating.value = false
  }
}

async function viewDocuments(kb: any) {
  currentKB.value = kb
  showDocDrawer.value = true
  docLoading.value = true
  try {
    const res = await fetchDocs(kb.id)
    documents.value = res.data.data.items || []
  } finally {
    docLoading.value = false
  }
}

async function handleArchiveDoc(docId: string) {
  await archiveDocument(currentKB.value.id, docId)
  ElMessage.success('文档已归档')
  await viewDocuments(currentKB.value)
}

function openAsk(kb: any) {
  currentKB.value = kb
  askQuery.value = ''
  askAnswer.value = ''
  showAskDialog.value = true
}

async function handleAsk() {
  if (!askQuery.value.trim()) return
  asking.value = true
  askAnswer.value = ''
  try {
    const res = await askKnowledgeBase(currentKB.value.id, { query: askQuery.value })
    askAnswer.value = res.data.data.answer || '未获取到回答'
  } finally {
    asking.value = false
  }
}

function statusTag(status: string) {
  if (status === 'done' || status === 'completed') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'failed') return 'danger'
  return 'info'
}

function formatTime(t: string) {
  return t ? new Date(t).toLocaleString('zh-CN') : '-'
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.ask-result {
  margin-top: 12px;
}
.answer-text {
  white-space: pre-wrap;
  line-height: 1.6;
  color: #303133;
  background: #f5f7fa;
  padding: 12px;
  border-radius: 6px;
}
</style>
