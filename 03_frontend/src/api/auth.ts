import request from './request'

// 登录
export function login(data: { email: string; password: string }) {
  return request.post('/auth/login', data)
}

// 注册
export function register(data: { email: string; password: string; nickname?: string }) {
  return request.post('/auth/register', data)
}

// 获取当前用户
export function getCurrentUser() {
  return request.get('/auth/me')
}
