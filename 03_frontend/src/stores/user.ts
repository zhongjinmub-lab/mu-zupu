import { defineStore } from 'pinia'
import { ref } from 'vue'
import { login as loginApi, getCurrentUser } from '@/api/auth'

// 用户状态管理
export const useUserStore = defineStore('user', () => {
  const token = ref(localStorage.getItem('token') || '')
  const tenantId = ref(localStorage.getItem('tenant_id') || '')
  const userInfo = ref<Record<string, any> | null>(null)

  // 登录
  async function login(email: string, password: string) {
    const res = await loginApi({ email, password })
    const data = res.data.data
    token.value = data.token
    tenantId.value = data.tenant_id || ''
    localStorage.setItem('token', data.token)
    if (data.tenant_id) {
      localStorage.setItem('tenant_id', data.tenant_id)
    }
    return data
  }

  // 获取用户信息
  async function fetchUser() {
    const res = await getCurrentUser()
    userInfo.value = res.data.data
    return userInfo.value
  }

  // 登出
  function logout() {
    token.value = ''
    tenantId.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('tenant_id')
  }

  // 切换租户
  function switchTenant(id: string) {
    tenantId.value = id
    localStorage.setItem('tenant_id', id)
  }

  return { token, tenantId, userInfo, login, fetchUser, logout, switchTenant }
})
