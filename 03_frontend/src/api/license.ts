import request from './request'

// License 列表
export function listLicenses() {
  return request.get('/licenses')
}

// 创建 License
export function createLicense(data: {
  type: string
  max_agents?: number
  expires_at?: string
  metadata?: Record<string, any>
}) {
  return request.post('/licenses', data)
}

// 激活 License
export function activateLicense(licenseId: string) {
  return request.post(`/licenses/${licenseId}/activate`)
}

// 吊销 License
export function revokeLicense(licenseId: string) {
  return request.post(`/licenses/${licenseId}/revoke`)
}

// 验证 License
export function verifyLicense(licenseId: string) {
  return request.post(`/licenses/${licenseId}/verify`)
}
