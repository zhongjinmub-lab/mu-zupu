<template>
  <div class="dashboard">
    <el-row :gutter="20" class="stat-row">
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>智能体</template>
          <div class="stat-value">{{ stats.agents }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>知识库</template>
          <div class="stat-value">{{ stats.knowledge_bases }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>会话总数</template>
          <div class="stat-value">{{ stats.conversations }}</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="hover">
          <template #header>License</template>
          <div class="stat-value">{{ stats.licenses }}</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="16">
        <el-card>
          <template #header>最近操作</template>
          <el-empty v-if="!recentActions.length" description="暂无数据" />
          <el-timeline v-else>
            <el-timeline-item
              v-for="item in recentActions"
              :key="item.id"
              :timestamp="item.created_at"
              placement="top"
            >
              {{ item.action }} - {{ item.resource_type }}
            </el-timeline-item>
          </el-timeline>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card>
          <template #header>系统状态</template>
          <el-descriptions :column="1" border size="small">
            <el-descriptions-item label="API">
              <el-tag type="success" size="small">正常</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="数据库">
              <el-tag type="success" size="small">正常</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Redis">
              <el-tag type="success" size="small">正常</el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import request from '@/api/request'

const stats = ref({
  agents: 0,
  knowledge_bases: 0,
  conversations: 0,
  licenses: 0,
})
const recentActions = ref<any[]>([])

onMounted(async () => {
  try {
    const res = await request.get('/analytics/summary')
    const data = res.data.data
    stats.value = {
      agents: data.resources?.agents || 0,
      knowledge_bases: data.resources?.knowledge_bases || 0,
      conversations: data.resources?.conversations || 0,
      licenses: data.resources?.licenses || 0,
    }
    recentActions.value = data.recent_actions || []
  } catch {
    // 静默处理，页面可展示空状态
  }
})
</script>

<style scoped>
.stat-value {
  font-size: 32px;
  font-weight: 700;
  color: #303133;
}

.stat-row .el-card {
  text-align: center;
}
</style>
