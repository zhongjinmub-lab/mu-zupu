<template>
  <div class="license-view">
    <div class="page-header">
      <h2>授权管理</h2>
      <el-button type="primary" @click="showCreateDialog = true">创建 License</el-button>
    </div>

    <!-- License 列表 -->
    <el-table :data="licenses" v-loading="loading" stripe>
      <el-table-column prop="code" label="License 编码" min-width="200" show-overflow-tooltip />
      <el-table-column prop="type" label="类型" width="100">
        <template #default="{ row }">
          <el-tag size="small">{{ row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="licenseStatusType(row.status)" size="small">
            {{ statusLabel(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="max_agents" label="最大 Agent" width="110" />
      <el-table-column prop="expires_at" label="到期时间" width="170">
        <template #default="{ row }">{{ formatTime(row.expires_at) }}</template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="170">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="row.status === 'inactive'"
            link type="success"
            @click="handleActivate(row.id)"
          >激活</el-button>
          <el-button
            v-if="row.status === 'active'"
            link type="warning"
            @click="handleVerify(row.id)"
          >验证</el-button>
          <el-button
            v-if="row.status === 'active'"
            link type="danger"
            @click="handleRevoke(row.id)"
          >吊销</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 创建 License 对话框 -->
    <el-dialog v-model="showCreateDialog" title="创建 License" width="480px">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="类型" required>
          <el-select v-model="createForm.type" placeholder="选择 License 类型">
            <el-option label="试用版" value="trial" />
            <el-option label="标准版" value="standard" />
            <el-option label="企业版" value="enterprise" />
          </el-select>
        </el-form-item>
        <el-form-item label="最大 Agent">
          <el-input-number v-model="createForm.max_agents" :min="1" :max="1000" />
        </el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker
            v-model="createForm.expires_at"
            type="datetime"
            placeholder="选择到期时间"
            value-format="YYYY-MM-DDTHH:mm:ssZ"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreateDialog = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="handleCreate">创建</el-button>
      </template>
    </el-dialog>

    <!-- 验证结果对话框 -->
    <el-dialog v-model="showVerifyDialog" title="License 验证结果" width="480px">
      <el-result
        :icon="verifyResult.valid ? 'success' : 'error'"
        :title="verifyResult.valid ? '验证通过' : '验证失败'"
        :sub-title="verifyResult.message"
      />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  listLicenses, createLicense, activateLicense, revokeLicense, verifyLicense
} from '@/api/license'

const licenses = ref<any[]>([])
const loading = ref(false)
const showCreateDialog = ref(false)
const creating = ref(false)
const showVerifyDialog = ref(false)
const verifyResult = ref({ valid: false, message: '' })

const createForm = ref({
  type: 'standard',
  max_agents: 10,
  expires_at: '',
})

onMounted(() => fetchList())

async function fetchList() {
  loading.value = true
  try {
    const res = await listLicenses()
    licenses.value = res.data.data.items || []
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!createForm.value.type) {
    ElMessage.warning('请选择 License 类型')
    return
  }
  creating.value = true
  try {
    await createLicense(createForm.value)
    ElMessage.success('License 创建成功')
    showCreateDialog.value = false
    createForm.value = { type: 'standard', max_agents: 10, expires_at: '' }
    await fetchList()
  } finally {
    creating.value = false
  }
}

async function handleActivate(id: string) {
  await ElMessageBox.confirm('确认激活此 License？', '激活确认')
  await activateLicense(id)
  ElMessage.success('License 已激活')
  await fetchList()
}

async function handleRevoke(id: string) {
  await ElMessageBox.confirm('确认吊销此 License？吊销后不可恢复。', '吊销确认', { type: 'warning' })
  await revokeLicense(id)
  ElMessage.success('License 已吊销')
  await fetchList()
}

async function handleVerify(id: string) {
  try {
    const res = await verifyLicense(id)
    verifyResult.value = {
      valid: res.data.data.valid !== false,
      message: res.data.data.message || '签名验证通过，License 有效',
    }
  } catch {
    verifyResult.value = { valid: false, message: '验证请求失败' }
  }
  showVerifyDialog.value = true
}

function licenseStatusType(status: string) {
  const map: Record<string, string> = {
    active: 'success',
    inactive: 'info',
    revoked: 'danger',
    expired: 'warning',
  }
  return map[status] || 'info'
}

function statusLabel(status: string) {
  const map: Record<string, string> = {
    active: '已激活',
    inactive: '未激活',
    revoked: '已吊销',
    expired: '已过期',
  }
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
</style>
