import { useCallback } from 'react'
import { useAuthStore } from '../stores/authStore'
import type { UserRole } from '../types/auth'

export function useAuth() {
  const user = useAuthStore((state) => state.user)
  const token = useAuthStore((state) => state.token)
  const logout = useAuthStore((state) => state.logout)

  const hasRole = useCallback((...roles: UserRole[]) => {
    if (!user) return false
    return roles.includes(user.role)
  }, [user])

  return {
    user,
    authenticated: Boolean(token && user),
    hasRole,
    logout
  }
}

