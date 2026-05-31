<template>
  <div class="analytics-view">
    <div class="page-header">
      <h2>数据分析</h2>
      <el-button @click="fetchSummary">刷新</el-button>
    </div>

    <!-- 资源统计卡片 -->
    <el-row :gutter="16" class="stat-row">
      <el-col :span="4" v-for="item in resourceCards" :key="item.label">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-label">{{ item.label }}</div>
          <div class="stat-value">{{ item.value }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <!-- 用量趋势 -->
      <el-col :span="16">
        <el-card>
          <template #header>近 7 天用量趋势</template>
          <el-table :data="usageTrend" stripe size="small">
            <el-table-column prop="date" label="日期" />
            <el-table-column prop="messages" label="消息数" />
            <el-table-column prop="searches" label="检索数" />
            <el-table-column prop="uploads" label="上传数" />
          </el-table>
        </el-card>
      </el-col>

      <!-- 风险项 -->
      <el-col :span="8">
        <el-card>
          <template #header>
            <span>风险项</span>
            <el-badge :value="riskItems.length" :max="9" class="risk-badge" />
          </template>
          <el-empty v-if="!riskItems.length" description="暂无风险" :image-size="60" />
          <ul v-else class="risk-list">
            <li v-for="(item, i) in riskItems" :key="i">
              <el-tag :type="item.level === 'high' ? 'danger' : 'warning'" size="small">
                {{ item.level }}
              </el-tag>
              <span>{{ item.description }}</span>
            </li>
          </ul>
        </el-card>
      </el-col>
    </el-row>

    <!-- 经营分布 -->
    <el-card style="margin-top: 20px;">
      <template #header>经营分布</template>
      <el-descriptions :column="3" border size="small">
        <el-descriptions-item label="活跃租户">{{ biz.active_tenants || 0 }}</el-descriptions-item>
        <el-descriptions-item label="付费租户">{{ biz.paid_tenants || 0 }}</el-descriptions-item>
        <el-descriptions-item label="总收入">{{ biz.total_revenue || '0.00' }}</el-descriptions-item>
        <el-descriptions-item label="活跃 License">{{ biz.active_licenses || 0 }}</el-descriptions-item>
        <el-descriptions-item label="过期 License">{{ biz.expired_licenses || 0 }}</el-descriptions-item>
        <el-descriptions-item label="订单总数">{{ biz.total_orders || 0 }}</el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import request from '@/api/request'

const summary = ref<any>({})
const loading = ref(false)

// 资源卡片数据
const resourceCards = computed(() => {
  const r = summary.value.resources || {}
  return [
    { label: '智能体', value: r.agents || 0 },
    { label: '知识库', value: r.knowledge_bases || 0 },
    { label: '文档', value: r.documents || 0 },
    { label: '会话', value: r.conversations || 0 },
    { label: '消息', value: r.messages || 0 },
    { label: 'License', value: r.licenses || 0 },
  ]
})

// 用量趋势
const usageTrend = computed(() => summary.value.usage_trend || [])

// 风险项
const riskItems = computed(() => summary.value.risk_items || [])

// 经营分布
const biz = computed(() => summary.value.business || {})

onMounted(() => fetchSummary())

async function fetchSummary() {
  loading.value = true
  try {
    const res = await request.get('/analytics/summary')
    summary.value = res.data.data || {}
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.stat-row .stat-card {
  text-align: center;
  padding: 8px;
}
.stat-label {
  font-size: 13px;
  color: #909399;
  margin-bottom: 4px;
}
.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: #303133;
}
.risk-badge {
  margin-left: 8px;
}
.risk-list {
  list-style: none;
  padding: 0;
  margin: 0;
}
.risk-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 13px;
}
</style>
