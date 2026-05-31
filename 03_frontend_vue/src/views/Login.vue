<template>
  <div class="login-page">
    <el-card class="login-card">
      <template #header>
        <h2>智能体族谱 SAAS 登录</h2>
      </template>
      <el-form :model="form" @submit.prevent="handleLogin" label-width="80px">
        <el-form-item label="邮箱">
          <el-input v-model="form.email" placeholder="请输入邮箱" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" native-type="submit" :loading="loading">登录</el-button>
        </el-form-item>
      </el-form>
      <p v-if="errorMsg" class="error-text">{{ errorMsg }}</p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import http from '../api'

const router = useRouter()
const loading = ref(false)
const errorMsg = ref('')
const form = ref({ email: '', password: '' })

async function handleLogin() {
  loading.value = true
  errorMsg.value = ''
  try {
    const data: any = await http.post('/auth/login', form.value)
    localStorage.setItem('token', data.token)
    if (data.tenants?.length) {
      localStorage.setItem('tenantId', data.tenants[0].id)
    }
    router.push('/dashboard')
  } catch (err: any) {
    errorMsg.value = err?.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background: #f5f7fa;
}
.login-card {
  width: 400px;
}
.error-text {
  color: #f56c6c;
  text-align: center;
  margin-top: 12px;
}
</style>
