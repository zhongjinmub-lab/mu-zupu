<template>
  <div class="genealogy-view">
    <div class="page-header">
      <h2>智能体族谱图</h2>
      <div>
        <el-input
          v-model="query"
          placeholder="搜索节点"
          style="width: 200px; margin-right: 12px;"
          clearable
          @clear="fetchGraph"
          @keyup.enter="fetchGraph"
        />
        <el-select v-model="relationType" placeholder="关系类型" clearable style="width: 140px;" @change="fetchGraph">
          <el-option label="全部" value="" />
          <el-option label="fork" value="fork" />
          <el-option label="inherit" value="inherit" />
          <el-option label="compose" value="compose" />
          <el-option label="route" value="route" />
        </el-select>
      </div>
    </div>

    <el-row :gutter="16" style="margin-bottom: 16px;">
      <el-col :span="6"><el-statistic title="节点数" :value="graph.summary?.nodes || 0" /></el-col>
      <el-col :span="6"><el-statistic title="边数" :value="graph.summary?.edges || 0" /></el-col>
      <el-col :span="6"><el-statistic title="根节点" :value="graph.summary?.roots || 0" /></el-col>
      <el-col :span="6"><el-statistic title="孤立节点" :value="graph.summary?.isolated || 0" /></el-col>
    </el-row>

    <el-card v-loading="loading">
      <template #header>族谱结构</template>
      <el-table :data="graph.edges" stripe size="small">
        <el-table-column prop="parent_name" label="父节点" min-width="120">
          <template #default="{ row }">{{ row.parent_name || '（根）' }}</template>
        </el-table-column>
        <el-table-column prop="child_name" label="子节点" min-width="120" />
        <el-table-column prop="relation_type" label="关系" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.relation_type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">{{ new Date(row.created_at).toLocaleString('zh-CN') }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getGenealogyGraph } from '@/api/agent'

const query = ref('')
const relationType = ref('')
const graph = ref<any>({ nodes: [], edges: [], summary: {} })
const loading = ref(false)

onMounted(() => fetchGraph())

async function fetchGraph() {
  loading.value = true
  try {
    const res = await getGenealogyGraph({
      q: query.value || undefined,
      relation_type: relationType.value || undefined,
    })
    graph.value = res.data.data
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
</style>
