<template>
  <div class="dashboard">
    <h2>总览</h2>
    <el-row :gutter="20">
      <el-col :span="6" v-for="card in cards" :key="card.label">
        <el-card shadow="hover">
          <template #header>{{ card.label }}</template>
          <p class="card-value">{{ card.value }}</p>
        </el-card>
      </el-col>
    </el-row>
    <el-card style="margin-top: 20px">
      <template #header>监控告警</template>
      <p v-if="alertLoading">加载中...</p>
      <div v-else>
        <el-tag :type="alertHealthy ? 'success' : 'danger'">{{ alertHealthy ? '健康' : '存在告警' }}</el-tag>
        <span style="margin-left: 12px">警告 {{ alertWarning }} / 严重 {{ alertCritical }}</span>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import http from '../api'

const cards = ref([
  { label: '智能体', value: '-' },
  { label: '知识库', value: '-' },
  { label: '工作流', value: '-' },
  { label: '渠道', value: '-' },
])
const alertLoading = ref(true)
const alertHealthy = ref(true)
const alertWarning = ref(0)
const alertCritical = ref(0)

onMounted(async () => {
  try {
    const [agents, kbs, workflows, channels, alertStatus] = await Promise.all([
      http.get('/agents'),
      http.get('/kbs'),
      http.get('/workflows'),
      http.get('/channels'),
      http.get('/settings/alert-status'),
    ])
    cards.value[0].value = String((agents as any)?.items?.length ?? 0)
    cards.value[1].value = String((kbs as any)?.items?.length ?? 0)
    cards.value[2].value = String((workflows as any)?.items?.length ?? 0)
    cards.value[3].value = String((channels as any)?.items?.length ?? 0)
    const ev = (alertStatus as any)?.evaluation
    if (ev) {
      alertHealthy.value = ev.healthy
      alertWarning.value = ev.warning_count || 0
      alertCritical.value = ev.critical_count || 0
    }
  } catch {
    // 静默忽略加载失败
  } finally {
    alertLoading.value = false
  }
})
</script>

<style scoped>
.dashboard { padding: 20px; }
.card-value { font-size: 28px; font-weight: bold; text-align: center; }
</style>
