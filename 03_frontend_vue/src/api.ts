import axios from 'axios'

// 创建 axios 实例，统一 baseURL 与认证头
const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api/v1',
  timeout: 30000,
})

// 请求拦截：自动附加 token 与 tenant-id
http.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  const tenantId = localStorage.getItem('tenantId')
  if (token) config.headers.Authorization = `Bearer ${token}`
  if (tenantId) config.headers['X-Tenant-ID'] = tenantId
  return config
})

// 响应拦截：提取 data 层、401 时跳登录
http.interceptors.response.use(
  (res) => res.data?.data ?? res.data,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = `${import.meta.env.BASE_URL}login`
    }
    return Promise.reject(err.response?.data ?? err)
  },
)

export default http
