<template>
  <div class="agent-list">
    <div class="page-header">
      <h2>智能体管理</h2>
      <el-button type="primary" @click="showCreateDialog = true">新建智能体</el-button>
    </div>

    <el-table :data="agents" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="120" />
      <el-table-column prop="code" label="编码" min-width="100" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="180">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="goDetail(row.id)">详情</el-button>
          <el-button link type="primary" @click="goVersions(row.id)">版本</el-button>
          <el-button link type="success" @click="handlePublish(row.id)">发布</el-button>
          <el-button link type="danger" @click="handleArchive(row.id)">归档</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 新建对话框 -->
    <el-dialog v-model="showCreateDialog" title="新建智能体" width="500px">
      <el-form :model="createForm" label-width="80px">
        <el-form-item label="名称" required>
          <el-input v-model="createForm.name" placeholder="智能体名称" />
        </el-form-item>
        <el-form-item label="编码" required>
          <el-input v-model="createForm.code" placeholder="唯一编码，小写字母+数字" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="createForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="系统提示">
          <el-input v-model="createForm.system_prompt" type="textarea" :rows="4" />
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
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listAgents, createAgent, publishAgent, archiveAgent } from '@/api/agent'

const router = useRouter()
const agents = ref<any[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const creating = ref(false)

const createForm = ref({
  name: '',
  code: '',
  description: '',
  system_prompt: '',
})

onMounted(() => fetchList())

async function fetchList() {
  loading.value = true
  try {
    const res = await listAgents()
    agents.value = res.data.data.items || []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  creating.value = true
  try {
    await createAgent(createForm.value)
    ElMessage.success('创建成功')
    showCreateDialog.value = false
    createForm.value = { name: '', code: '', description: '', system_prompt: '' }
    await fetchList()
  } finally {
    creating.value = false
  }
}

async function handlePublish(id: string) {
  await ElMessageBox.confirm('确认发布该智能体？', '发布确认')
  await publishAgent(id)
  ElMessage.success('已发布')
  await fetchList()
}

async function handleArchive(id: string) {
  await ElMessageBox.confirm('确认归档该智能体？归档后不可恢复。', '归档确认', { type: 'warning' })
  await archiveAgent(id)
  ElMessage.success('已归档')
  await fetchList()
}

function goDetail(id: string) {
  router.push(`/agents/${id}`)
}

function goVersions(id: string) {
  router.push(`/agents/${id}/versions`)
}

function statusType(status: string) {
  const map: Record<string, string> = { published: 'success', draft: 'info', archived: 'danger' }
  return map[status] || 'info'
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
</style>
