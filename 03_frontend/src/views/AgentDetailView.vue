<template>
  <div class="agent-detail" v-loading="loading">
    <div class="page-header">
      <h2>{{ agent.name || '智能体详情' }}</h2>
      <div>
        <el-button @click="$router.push('/agents')">返回列表</el-button>
        <el-button type="primary" @click="$router.push(`/agents/${agentId}/versions`)">版本管理</el-button>
      </div>
    </div>

    <el-descriptions :column="2" border>
      <el-descriptions-item label="名称">{{ agent.name }}</el-descriptions-item>
      <el-descriptions-item label="编码">{{ agent.code }}</el-descriptions-item>
      <el-descriptions-item label="状态">
        <el-tag :type="agent.status === 'published' ? 'success' : 'info'" size="small">
          {{ agent.status }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="创建时间">{{ formatTime(agent.created_at) }}</el-descriptions-item>
      <el-descriptions-item label="描述" :span="2">{{ agent.description || '无' }}</el-descriptions-item>
      <el-descriptions-item label="系统提示词" :span="2">
        <pre class="prompt-text">{{ agent.system_prompt || '未配置' }}</pre>
      </el-descriptions-item>
    </el-descriptions>

    <!-- 绑定知识库 -->
    <el-card style="margin-top: 20px;">
      <template #header>已绑定知识库</template>
      <el-empty v-if="!bindings.length" description="暂未绑定知识库" />
      <el-tag v-for="b in bindings" :key="b.id" style="margin-right: 8px;">
        {{ b.knowledge_base || b.knowledge_base_id }}
      </el-tag>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getAgent } from '@/api/agent'
import request from '@/api/request'

const route = useRoute()
const agentId = route.params.agentId as string
const agent = ref<any>({})
const bindings = ref<any[]>([])
const loading = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    const res = await getAgent(agentId)
    agent.value = res.data.data
    const kbRes = await request.get(`/agents/${agentId}/knowledge-bases`)
    bindings.value = kbRes.data.data.items || []
  } finally {
    loading.value = false
  }
})

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

.prompt-text {
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
  font-size: 13px;
  color: #606266;
}
</style>
