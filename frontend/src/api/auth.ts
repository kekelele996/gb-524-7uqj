import { apiClient } from './client'
import type { LoginResult, User } from '../types/auth'

export const authApi = {
  login: (email: string, password: string) => apiClient.post<LoginResult>('/auth/login', { email, password }),
  me: () => apiClient.get<User>('/auth/me')
}

