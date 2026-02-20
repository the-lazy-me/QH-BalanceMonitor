import axios from 'axios'

// 使用相对路径，通过 Next.js rewrites 代理到后端
// 在浏览器中会请求到前端服务器的 /api，然后由 Next.js 转发到后端
const API_BASE_URL = '/api'

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Token 管理
const TOKEN_KEY = 'auth_token'

export const setToken = (token: string) => {
  if (typeof window !== 'undefined') {
    localStorage.setItem(TOKEN_KEY, token)
  }
}

export const getToken = (): string | null => {
  if (typeof window !== 'undefined') {
    return localStorage.getItem(TOKEN_KEY)
  }
  return null
}

export const removeToken = () => {
  if (typeof window !== 'undefined') {
    localStorage.removeItem(TOKEN_KEY)
  }
}

// 请求拦截器 - 添加 token
api.interceptors.request.use(
  (config) => {
    const token = getToken()
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器 - 处理认证错误
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // 未授权，清除 token 并跳转到登录页
      removeToken()
      if (typeof window !== 'undefined' && !window.location.pathname.includes('/login')) {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

// 认证相关
export const checkInitStatus = () => api.get('/auth/status')
export const initializeSystem = (username: string, password: string) => 
  api.post('/auth/initialize', { username, password })
export const login = (username: string, password: string) => 
  api.post('/auth/login', { username, password })
export const changePassword = (oldPassword: string, newPassword: string) =>
  api.post('/auth/change-password', { old_password: oldPassword, new_password: newPassword })

// 供应商相关
export const getProviders = () => api.get('/providers')
export const createProvider = (data: any) => api.post('/providers', data)
export const updateProvider = (id: number, data: any) => api.put(`/providers/${id}`, data)
export const deleteProvider = (id: number) => api.delete(`/providers/${id}`)

// 余额相关
export const getCurrentBalance = () => api.get('/balance')
export const getBalanceHistory = (providerId?: number, limit?: number, days?: number) => {
  const params: any = {}
  if (providerId) params.provider_id = providerId
  if (limit) params.limit = limit
  if (days) params.days = days
  return api.get('/balance/history', { params })
}
export const manualCheckBalance = () => api.post('/balance/check')

// SMTP 设置
export const getSMTPSettings = () => api.get('/smtp')
export const updateSMTPSettings = (data: any) => api.put('/smtp', data)
export const testSMTP = () => api.post('/smtp/test')

// 监控设置
export const getMonitorSettings = () => api.get('/monitor')
export const updateMonitorSettings = (data: any) => api.put('/monitor', data)

// 通知历史
export const getNotifications = (limit?: number) => {
  const params: any = {}
  if (limit) params.limit = limit
  return api.get('/notifications', { params })
}

// 记账相关
export const getAccountingRecords = (params?: {
  provider_id?: number; start_date?: string; end_date?: string
}) => api.get('/accounting', { params })

export const createAccountingRecord = (data: {
  provider_id: number; amount: number; date: string; note?: string
}) => api.post('/accounting', data)

export const updateAccountingRecord = (id: number, data: {
  amount?: number; date?: string; note?: string
}) => api.put(`/accounting/${id}`, data)

export const deleteAccountingRecord = (id: number) =>
  api.delete(`/accounting/${id}`)

export const getProviderCostStats = (params?: {
  start_date?: string; end_date?: string
}) => api.get('/accounting/stats/provider', { params })

export const getMonthlyCostStats = (params?: {
  provider_id?: number
}) => api.get('/accounting/stats/monthly', { params })

export default api

