<template>
  <div class="agent-versions">
    <div class="page-header">
      <h2>版本管理</h2>
      <div>
        <el-button @click="$router.push(`/agents/${agentId}`)">返回详情</el-button>
        <el-button type="primary" @click="showCreateDialog = true">创建版本</el-button>
      </div>
    </div>

    <el-table :data="versions" v-loading="loading" stripe>
      <el-table-column prop="version_no" label="版本号" width="120" />
      <el-table-column prop="channel" label="渠道" width="100">
        <template #default="{ row }">
          <el-tag size="small">{{ row.channel }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="versionStatusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="publish_note" label="发布说明" min-width="200" show-overflow-tooltip />
      <el-table-column prop="created_at" label="创建时间" width="170">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column prop="published_at" label="发布时间" width="170">
        <template #default="{ row }">{{ formatTime(row.published_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="row.status === 'draft'"
            link type="success"
            @click="handlePublish(row.id)"
          >发布</el-button>
          <el-button
            v-if="row.status === 'published'"
            link type="warning"
            @click="handleRollback(row.id)"
          >回滚</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 创建版本对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建版本" width="520px">
      <el-form :model="createForm" label-width="90px">
        <el-form-item label="版本号" required>
          <el-input v-model="createForm.version_no" placeholder="如 v1.0.0" />
        </el-form-item>
        <el-form-item label="渠道">
          <el-select v-model="createForm.channel" placeholder="选择发布渠道">
            <el-option label="Web" value="web" />
            <el-option label="微信公众号" value="wechat" />
            <el-option label="小程序" value="miniapp" />
            <el-option label="API" value="api" />
            <el-option label="H5" value="h5" />
            <el-option label="企业微信" value="enterprise_wechat" />
          </el-select>
        </el-form-item>
        <el-form-item label="提示词">
          <el-input v-model="createForm.prompt" type="textarea" :rows="4" placeholder="版本系统提示词快照" />
        </el-form-item>
        <el-form-item label="发布说明">
          <el-input v-model="createForm.publish_note" type="textarea" :rows="2" placeholder="本次版本变更说明" />
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
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listVersions, createVersion, publishVersion, rollbackVersion } from '@/api/agent'

const route = useRoute()
const agentId = route.params.agentId as string
const versions = ref<any[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const creating = ref(false)

const createForm = ref({
  version_no: '',
  channel: 'web',
  prompt: '',
  publish_note: '',
})

onMounted(() => fetchVersions())

async function fetchVersions() {
  loading.value = true
  try {
    const res = await listVersions(agentId)
    versions.value = res.data.data.items || []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!createForm.value.version_no) {
    ElMessage.warning('请输入版本号')
    return
  }
  creating.value = true
  try {
    await createVersion(agentId, createForm.value)
    ElMessage.success('版本创建成功')
    showCreateDialog.value = false
    createForm.value = { version_no: '', channel: 'web', prompt: '', publish_note: '' }
    await fetchVersions()
  } finally {
    creating.value = false
  }
}

async function handlePublish(versionId: string) {
  await ElMessageBox.confirm('确认发布该版本？发布后当前已发布版本将自动归档。', '发布确认')
  await publishVersion(agentId, versionId)
  ElMessage.success('版本已发布')
  await fetchVersions()
}

async function handleRollback(versionId: string) {
  await ElMessageBox.confirm('确认回滚该版本？', '回滚确认', { type: 'warning' })
  await rollbackVersion(agentId, versionId)
  ElMessage.success('版本已回滚')
  await fetchVersions()
}

function versionStatusType(status: string) {
  const map: Record<string, string> = {
    published: 'success',
    draft: 'info',
    archived: 'warning',
    rollback: 'danger',
  }
  return map[status] || 'info'
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    published: '已发布',
    draft: '草稿',
    archived: '已归档',
    rollback: '已回滚',
  }
  return map[status] || status
}

function formatTime(t: string | null) {
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
