export interface ApiResponse<T> {
  data: T
  request_id: string
}

export interface PageMeta {
  page: number
  page_size: number
  total: number
}

export interface PageResponse<T> extends ApiResponse<T> {
  meta: PageMeta
}

export interface ApiErrorPayload {
  error: {
    code: string
    message: string
    details?: Record<string, unknown>
  }
  request_id: string
}

