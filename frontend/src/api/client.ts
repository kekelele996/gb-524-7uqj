import type { ApiErrorPayload, ApiResponse, PageResponse } from '../types/api'

export class ApiError extends Error {
  readonly code: string
  readonly status: number
  readonly requestId: string
  readonly details?: Record<string, unknown>

  constructor(status: number, payload: ApiErrorPayload) {
    super(payload.error.message)
    this.name = 'ApiError'
    this.code = payload.error.code
    this.status = status
    this.requestId = payload.request_id
    this.details = payload.error.details
  }
}

const TOKEN_KEY = 'spectrum_token'

export const session = {
  token: () => sessionStorage.getItem(TOKEN_KEY),
  setToken: (token: string) => sessionStorage.setItem(TOKEN_KEY, token),
  clear: () => sessionStorage.removeItem(TOKEN_KEY)
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  const token = session.token()
  if (token) headers.set('Authorization', `Bearer ${token}`)

  let response: Response
  try {
    response = await fetch(`/api/v1${path}`, { ...init, headers })
  } catch {
    const detail = '无法连接分析服务，请检查网络和服务状态。'
    window.dispatchEvent(new CustomEvent('api:error', { detail }))
    throw new Error(detail)
  }

  const payload = await response.json()
  if (!response.ok) {
    const error = new ApiError(response.status, payload as ApiErrorPayload)
    if (response.status === 401 && path !== '/auth/login') {
      session.clear()
      window.dispatchEvent(new Event('auth:expired'))
    }
    window.dispatchEvent(new CustomEvent('api:error', { detail: `${error.code}：${error.message}` }))
    throw error
  }
  return payload as T
}

export const apiClient = {
  get: <T>(path: string) => request<ApiResponse<T>>(path),
  getPage: <T>(path: string) => request<PageResponse<T>>(path),
  post: <T>(path: string, body: unknown) => request<ApiResponse<T>>(path, { method: 'POST', body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) => request<ApiResponse<T>>(path, { method: 'PUT', body: JSON.stringify(body) })
}

