<template>
  <div class="settings-view">
    <div class="page-header">
      <h2>系统设置</h2>
      <el-button @click="refreshAll">刷新全部</el-button>
    </div>

    <el-tabs v-model="activeTab">
      <!-- 限流策略 -->
      <el-tab-pane label="限流策略" name="rateLimit">
        <el-card v-loading="loadingRL">
          <el-descriptions :column="2" border size="small" v-if="rateLimit">
            <el-descriptions-item label="全局限流">
              {{ rateLimit.global_rps || '-' }} RPS
            </el-descriptions-item>
            <el-descriptions-item label="租户限流">
              {{ rateLimit.tenant_rps || '-' }} RPS
            </el-descriptions-item>
            <el-descriptions-item label="用户限流">
              {{ rateLimit.user_rps || '-' }} RPS
            </el-descriptions-item>
            <el-descriptions-item label="存储后端">
              {{ rateLimit.backend || 'memory' }}
            </el-descriptions-item>
          </el-descriptions>
          <div v-if="rateLimit?.groups" style="margin-top:16px;">
            <h4>分组限流</h4>
            <el-table :data="rateLimit.groups" stripe size="small">
              <el-table-column prop="name" label="分组" />
              <el-table-column prop="rps" label="RPS" width="80" />
              <el-table-column prop="burst" label="Burst" width="80" />
              <el-table-column prop="pattern" label="匹配规则" min-width="200" />
            </el-table>
          </div>
        </el-card>
      </el-tab-pane>

      <!-- 运行配置 -->
      <el-tab-pane label="运行配置" name="runtime">
        <el-card v-loading="loadingRT">
          <el-descriptions :column="2" border size="small" v-if="runtime">
            <el-descriptions-item
              v-for="(value, key) in runtime" :key="key"
              :label="String(key)"
            >
              <template v-if="typeof value === 'boolean'">
                <el-tag :type="value ? 'success' : 'info'" size="small">{{ value }}</el-tag>
              </template>
              <template v-else>{{ value }}</template>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-tab-pane>

      <!-- 监控 -->
      <el-tab-pane label="监控" name="monitoring">
        <el-card v-loading="loadingMon">
          <el-descriptions :column="2" border size="small" v-if="monitoring">
            <el-descriptions-item
              v-for="(value, key) in monitoring" :key="key"
              :label="String(key)"
            >{{ typeof value === 'object' ? JSON.stringify(value) : value }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-tab-pane>

      <!-- 向量检索健康 -->
      <el-tab-pane label="向量检索" name="vectorSearch">
        <el-card v-loading="loadingVS">
          <el-descriptions :column="2" border size="small" v-if="vectorSearch">
            <el-descriptions-item
              v-for="(value, key) in vectorSearch" :key="key"
              :label="String(key)"
            >{{ typeof value === 'object' ? JSON.stringify(value) : value }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-tab-pane>

      <!-- 敏感字段 -->
      <el-tab-pane label="敏感字段" name="sensitive">
        <el-card v-loading="loadingSF">
          <el-descriptions :column="1" border size="small" v-if="sensitiveFields">
            <el-descriptions-item
              v-for="(value, key) in sensitiveFields" :key="key"
              :label="String(key)"
            >{{ typeof value === 'object' ? JSON.stringify(value) : value }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-tab-pane>

      <!-- 告警 -->
      <el-tab-pane label="告警" name="alert">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-card v-loading="loadingAlert">
              <template #header>告警策略</template>
              <el-descriptions :column="1" border size="small" v-if="alertPolicy">
                <el-descriptions-item
                  v-for="(value, key) in alertPolicy" :key="key"
                  :label="String(key)"
                >{{ typeof value === 'object' ? JSON.stringify(value) : value }}</el-descriptions-item>
              </el-descriptions>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card v-loading="loadingAlert">
              <template #header>告警状态</template>
              <el-descriptions :column="1" border size="small" v-if="alertStatus">
                <el-descriptions-item
                  v-for="(value, key) in alertStatus" :key="key"
                  :label="String(key)"
                >{{ typeof value === 'object' ? JSON.stringify(value) : value }}</el-descriptions-item>
              </el-descriptions>
            </el-card>
          </el-col>
        </el-row>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  getRateLimitPolicy, getRuntimeSummary, getMonitoringSnapshot,
  getSensitiveFieldSummary, getVectorSearchSummary,
  getAlertPolicy, getAlertStatus
} from '@/api/settings'

const activeTab = ref('rateLimit')

const rateLimit = ref<any>(null)
const runtime = ref<any>(null)
const monitoring = ref<any>(null)
const vectorSearch = ref<any>(null)
const sensitiveFields = ref<any>(null)
const alertPolicy = ref<any>(null)
const alertStatus = ref<any>(null)

const loadingRL = ref(false)
const loadingRT = ref(false)
const loadingMon = ref(false)
const loadingVS = ref(false)
const loadingSF = ref(false)
const loadingAlert = ref(false)

onMounted(() => refreshAll())

async function refreshAll() {
  fetchRateLimit()
  fetchRuntime()
  fetchMonitoring()
  fetchVectorSearch()
  fetchSensitiveFields()
  fetchAlerts()
}

async function fetchRateLimit() {
  loadingRL.value = true
  try {
    const res = await getRateLimitPolicy()
    rateLimit.value = res.data.data
  } catch { /* 静默 */ }
  finally { loadingRL.value = false }
}

async function fetchRuntime() {
  loadingRT.value = true
  try {
    const res = await getRuntimeSummary()
    runtime.value = res.data.data
  } catch { /* 静默 */ }
  finally { loadingRT.value = false }
}

async function fetchMonitoring() {
  loadingMon.value = true
  try {
    const res = await getMonitoringSnapshot()
    monitoring.value = res.data.data
  } catch { /* 静默 */ }
  finally { loadingMon.value = false }
}

async function fetchVectorSearch() {
  loadingVS.value = true
  try {
    const res = await getVectorSearchSummary()
    vectorSearch.value = res.data.data
  } catch { /* 静默 */ }
  finally { loadingVS.value = false }
}

async function fetchSensitiveFields() {
  loadingSF.value = true
  try {
    const res = await getSensitiveFieldSummary()
    sensitiveFields.value = res.data.data
  } catch { /* 静默 */ }
  finally { loadingSF.value = false }
}

async function fetchAlerts() {
  loadingAlert.value = true
  try {
    const [pRes, sRes] = await Promise.all([getAlertPolicy(), getAlertStatus()])
    alertPolicy.value = pRes.data.data
    alertStatus.value = sRes.data.data
  } catch { /* 静默 */ }
  finally { loadingAlert.value = false }
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
