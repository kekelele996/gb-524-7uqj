export type UserRole = 'observer' | 'analyst' | 'reviewer' | 'admin'

export interface User {
  id: number
  email: string
  display_name: string
  role: UserRole
  active: boolean
  created_at: string
}

export interface LoginResult {
  token: string
  user: User
}

