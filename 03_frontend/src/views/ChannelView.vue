<template>
  <div class="channel-view">
    <div class="page-header">
      <h2>渠道接入</h2>
      <el-button type="primary" @click="openCreate">新建渠道</el-button>
    </div>

    <!-- 渠道概览统计 -->
    <el-row :gutter="16" class="summary-row" v-if="summary">
      <el-col :span="6"><el-statistic title="渠道总数" :value="summary.total || 0" /></el-col>
      <el-col :span="6"><el-statistic title="已启用" :value="summary.enabled || 0" /></el-col>
      <el-col :span="6"><el-statistic title="已禁用" :value="summary.disabled || 0" /></el-col>
      <el-col :span="6">
        <el-statistic title="类型数" :value="Object.keys(summary.by_type || {}).length" />
      </el-col>
    </el-row>

    <!-- 渠道列表 -->
    <el-table :data="channels" v-loading="loading" stripe style="margin-top: 16px;">
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="type" label="类型" width="130">
        <template #default="{ row }">
          <el-tag size="small">{{ typeLabel(row.type) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="channel_key" label="渠道 Key" min-width="200" show-overflow-tooltip />
      <el-table-column prop="status" label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="170">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="320" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="showEmbed(row)">接入代码</el-button>
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link type="primary" @click="handleDuplicate(row.id)">复制</el-button>
          <el-button
            v-if="row.status !== 'enabled'"
            link type="success"
            @click="handleEnable(row.id)"
          >启用</el-button>
          <el-button
            v-else
            link type="warning"
            @click="handleDisable(row.id)"
          >禁用</el-button>
          <el-popconfirm title="确认归档此渠道？" @confirm="handleArchive(row.id)">
            <template #reference>
              <el-button link type="danger">归档</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <!-- 创建/编辑对话框 -->
    <el-dialog v-model="showDialog" :title="editingId ? '编辑渠道' : '新建渠道'" width="520px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="渠道名称" />
        </el-form-item>
        <el-form-item label="绑定 Agent" required v-if="!editingId">
          <el-select v-model="form.agent_id" placeholder="选择已发布的 Agent" filterable>
            <el-option
              v-for="a in agents" :key="a.id"
              :label="a.name" :value="a.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="渠道类型" required v-if="!editingId">
          <el-select v-model="form.type" placeholder="选择渠道类型">
            <el-option
              v-for="t in installableTypes" :key="t.type"
              :label="t.name" :value="t.type"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showDialog = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ editingId ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 接入代码对话框 -->
    <el-dialog v-model="showEmbedDialog" title="渠道接入代码" width="640px">
      <el-descriptions :column="1" border size="small" v-if="embed">
        <el-descriptions-item label="渠道 Key">{{ embed.channel_key }}</el-descriptions-item>
        <el-descriptions-item label="类型">{{ typeLabel(embed.type) }}</el-descriptions-item>
        <el-descriptions-item label="启用状态">
          <el-tag :type="embed.enabled ? 'success' : 'info'" size="small">
            {{ embed.enabled ? '已启用' : '未启用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="API 端点">{{ embed.api_endpoint }}</el-descriptions-item>
      </el-descriptions>
      <div v-if="embed?.embed_snippet" class="snippet-block">
        <div class="snippet-head">
          <span>接入代码</span>
          <el-button link type="primary" size="small" @click="copySnippet">复制</el-button>
        </div>
        <pre class="snippet-code">{{ embed.embed_snippet }}</pre>
      </div>
      <ul class="instructions" v-if="embed?.instructions?.length">
        <li v-for="(line, i) in embed.instructions" :key="i">{{ line }}</li>
      </ul>
    </el-dialog>
  </div>
</template>


<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  listChannels, listChannelTypes, channelsSummary, getChannelEmbed,
  createChannel, updateChannel, duplicateChannel,
  enableChannel, disableChannel, archiveChannel
} from '@/api/channel'
import { listAgents } from '@/api/agent'

const channels = ref<any[]>([])
const channelTypes = ref<any[]>([])
const agents = ref<any[]>([])
const summary = ref<any>(null)
const loading = ref(false)

// 表单
const showDialog = ref(false)
const submitting = ref(false)
const editingId = ref('')
const form = ref({ name: '', agent_id: '', type: '' })

// 接入代码
const showEmbedDialog = ref(false)
const embed = ref<any>(null)

// 仅展示可安装的渠道类型
const installableTypes = computed(() => channelTypes.value.filter((t) => t.installable))

onMounted(() => {
  fetchChannels()
  fetchTypes()
  fetchSummary()
  fetchAgents()
})

async function fetchChannels() {
  loading.value = true
  try {
    const res = await listChannels()
    channels.value = res.data.data.items || []
  } finally {
    loading.value = false
  }
}

async function fetchTypes() {
  try {
    const res = await listChannelTypes()
    channelTypes.value = res.data.data.items || []
  } catch { /* 静默 */ }
}

async function fetchSummary() {
  try {
    const res = await channelsSummary()
    summary.value = res.data.data || {}
  } catch { /* 静默 */ }
}

async function fetchAgents() {
  try {
    const res = await listAgents()
    agents.value = res.data.data.items || []
  } catch { /* 静默 */ }
}

function openCreate() {
  editingId.value = ''
  form.value = { name: '', agent_id: '', type: '' }
  showDialog.value = true
}

function openEdit(row: any) {
  editingId.value = row.id
  form.value = { name: row.name, agent_id: row.agent_id, type: row.type }
  showDialog.value = true
}

async function handleSubmit() {
  if (!form.value.name) {
    ElMessage.warning('请填写渠道名称')
    return
  }
  submitting.value = true
  try {
    if (editingId.value) {
      await updateChannel(editingId.value, { name: form.value.name })
      ElMessage.success('渠道已更新')
    } else {
      if (!form.value.agent_id || !form.value.type) {
        ElMessage.warning('请选择绑定 Agent 和渠道类型')
        return
      }
      await createChannel(form.value)
      ElMessage.success('渠道创建成功')
    }
    showDialog.value = false
    await Promise.all([fetchChannels(), fetchSummary()])
  } finally {
    submitting.value = false
  }
}

async function handleDuplicate(id: string) {
  await duplicateChannel(id)
  ElMessage.success('渠道已复制')
  await Promise.all([fetchChannels(), fetchSummary()])
}

async function handleEnable(id: string) {
  await enableChannel(id)
  ElMessage.success('渠道已启用')
  await Promise.all([fetchChannels(), fetchSummary()])
}

async function handleDisable(id: string) {
  await disableChannel(id)
  ElMessage.success('渠道已禁用')
  await Promise.all([fetchChannels(), fetchSummary()])
}

async function handleArchive(id: string) {
  await archiveChannel(id)
  ElMessage.success('渠道已归档')
  await Promise.all([fetchChannels(), fetchSummary()])
}

async function showEmbed(row: any) {
  const res = await getChannelEmbed(row.id)
  embed.value = res.data.data
  showEmbedDialog.value = true
}

async function copySnippet() {
  if (embed.value?.embed_snippet) {
    await navigator.clipboard.writeText(embed.value.embed_snippet)
    ElMessage.success('已复制接入代码')
  }
}

// 类型中文标签
function typeLabel(type: string) {
  const found = channelTypes.value.find((t) => t.type === type)
  return found?.name || type
}

function statusType(status: string) {
  const map: Record<string, string> = { enabled: 'success', disabled: 'info', archived: 'danger' }
  return map[status] || 'info'
}

function statusLabel(status: string) {
  const map: Record<string, string> = { enabled: '已启用', disabled: '已禁用', archived: '已归档' }
  return map[status] || status
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
.summary-row {
  margin-bottom: 8px;
}
.snippet-block {
  margin-top: 16px;
}
.snippet-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
  font-size: 13px;
  color: #606266;
}
.snippet-code {
  background: #1d1e2c;
  color: #a9e34b;
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
  font-size: 12px;
  margin: 0;
}
.instructions {
  margin-top: 12px;
  padding-left: 20px;
  font-size: 13px;
  color: #606266;
  line-height: 1.8;
}
</style>
