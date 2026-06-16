<template>
  <div class="webhook-view">
    <div class="page-header">
      <h2>Webhook 管理</h2>
      <el-button type="primary" @click="showCreateDialog = true">新建端点</el-button>
    </div>

    <!-- 投递摘要 -->
    <el-row :gutter="16" class="summary-row" v-if="summary">
      <el-col :span="6">
        <el-statistic title="总投递" :value="summary.total || 0" />
      </el-col>
      <el-col :span="6">
        <el-statistic title="成功" :value="summary.success || 0" />
      </el-col>
      <el-col :span="6">
        <el-statistic title="失败" :value="summary.failed || 0" />
      </el-col>
      <el-col :span="6">
        <el-statistic title="待重试" :value="summary.pending_retry || 0" />
      </el-col>
    </el-row>

    <!-- Webhook 端点列表 -->
    <el-card style="margin-top: 16px;">
      <template #header>端点配置</template>
      <el-table :data="webhooks" v-loading="loading" stripe size="small">
        <el-table-column prop="url" label="URL" min-width="240" show-overflow-tooltip />
        <el-table-column prop="events" label="事件" min-width="180">
          <template #default="{ row }">
            <el-tag v-for="e in (row.events || [])" :key="e" size="small" style="margin: 2px;">
              {{ e }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleTest(row.id)">测试</el-button>
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm title="确认删除此端点？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 投递记录 -->
    <el-card style="margin-top: 20px;">
      <template #header>
        <div style="display:flex;justify-content:space-between;align-items:center;">
          <span>投递记录</span>
          <el-select v-model="deliveryFilter" placeholder="状态筛选" clearable size="small"
            style="width:120px;" @change="fetchDeliveries">
            <el-option label="全部" value="" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="待重试" value="pending" />
          </el-select>
        </div>
      </template>
      <el-table :data="deliveries" v-loading="deliveryLoading" stripe size="small">
        <el-table-column prop="webhook_url" label="目标 URL" min-width="200" show-overflow-tooltip />
        <el-table-column prop="event" label="事件" width="160" />
        <el-table-column prop="status" label="状态" width="90">
          <template #default="{ row }">
            <el-tag :type="deliveryStatusType(row.status)" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="http_code" label="HTTP" width="70" />
        <el-table-column prop="attempts" label="尝试" width="60" />
        <el-table-column prop="created_at" label="时间" width="160">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="80">
          <template #default="{ row }">
            <el-button
              v-if="row.status === 'failed'"
              link type="warning" size="small"
              @click="handleRetry(row.id)"
            >重试</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog v-model="showCreateDialog" :title="editingId ? '编辑端点' : '新建端点'" width="520px">
      <el-form :model="form" label-width="80px">
        <el-form-item label="URL" required>
          <el-input v-model="form.url" placeholder="https://example.com/webhook" />
        </el-form-item>
        <el-form-item label="事件">
          <el-select v-model="form.events" multiple placeholder="选择订阅事件">
            <el-option label="agent.chat.finished" value="agent.chat.finished" />
            <el-option label="document.parsed" value="document.parsed" />
            <el-option label="document.embedded" value="document.embedded" />
            <el-option label="license.activated" value="license.activated" />
            <el-option label="order.paid" value="order.paid" />
          </el-select>
        </el-form-item>
        <el-form-item label="Secret">
          <el-input v-model="form.secret" placeholder="用于验签的密钥（可选）" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeDialog">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ editingId ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  listWebhooks, createWebhook, updateWebhook, deleteWebhook,
  testWebhook, listDeliveries as fetchDeliveriesApi,
  deliverySummary as fetchSummaryApi, retryDelivery
} from '@/api/webhook'

const webhooks = ref<any[]>([])
const loading = ref(false)
const summary = ref<any>(null)

// 投递记录
const deliveries = ref<any[]>([])
const deliveryLoading = ref(false)
const deliveryFilter = ref('')

// 表单
const showCreateDialog = ref(false)
const submitting = ref(false)
const editingId = ref('')
const form = ref({ url: '', events: [] as string[], secret: '', description: '' })

onMounted(() => {
  fetchWebhooks()
  fetchSummary()
  fetchDeliveries()
})

async function fetchWebhooks() {
  loading.value = true
  try {
    const res = await listWebhooks()
    webhooks.value = res.data.data.items || []
  } finally {
    loading.value = false
  }
}

async function fetchSummary() {
  try {
    const res = await fetchSummaryApi()
    summary.value = res.data.data || {}
  } catch { /* 静默 */ }
}

async function fetchDeliveries() {
  deliveryLoading.value = true
  try {
    const params: any = { limit: 50 }
    if (deliveryFilter.value) params.status = deliveryFilter.value
    const res = await fetchDeliveriesApi(params)
    deliveries.value = res.data.data.items || []
  } finally {
    deliveryLoading.value = false
  }
}

async function handleSubmit() {
  if (!form.value.url) {
    ElMessage.warning('请填写 URL')
    return
  }
  submitting.value = true
  try {
    if (editingId.value) {
      await updateWebhook(editingId.value, form.value)
      ElMessage.success('端点已更新')
    } else {
      await createWebhook(form.value)
      ElMessage.success('端点创建成功')
    }
    closeDialog()
    await fetchWebhooks()
  } finally {
    submitting.value = false
  }
}

function openEdit(row: any) {
  editingId.value = row.id
  form.value = {
    url: row.url || '',
    events: row.events || [],
    secret: '',
    description: row.description || '',
  }
  showCreateDialog.value = true
}

function closeDialog() {
  showCreateDialog.value = false
  editingId.value = ''
  form.value = { url: '', events: [], secret: '', description: '' }
}

async function handleDelete(id: string) {
  await deleteWebhook(id)
  ElMessage.success('端点已删除')
  await fetchWebhooks()
}

async function handleTest(id: string) {
  await testWebhook(id)
  ElMessage.success('测试投递已发送')
  setTimeout(() => fetchDeliveries(), 1000)
}

async function handleRetry(id: string) {
  await retryDelivery(id)
  ElMessage.success('已触发重试')
  setTimeout(() => { fetchDeliveries(); fetchSummary() }, 1000)
}

function deliveryStatusType(status: string) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  return 'warning'
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
</style>
