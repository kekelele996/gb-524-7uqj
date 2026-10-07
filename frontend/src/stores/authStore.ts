import { create } from 'zustand'
import { authApi } from '../api/auth'
import { session } from '../api/client'
import type { User } from '../types/auth'

interface AuthState {
  user: User | null
  token: string | null
  busy: boolean
  initialized: boolean
  login: (email: string, password: string) => Promise<void>
  loadMe: () => Promise<void>
  logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: session.token(),
  busy: false,
  initialized: false,
  login: async (email, password) => {
    set({ busy: true })
    try {
      const response = await authApi.login(email, password)
      session.setToken(response.data.token)
      set({ user: response.data.user, token: response.data.token, initialized: true })
    } finally {
      set({ busy: false })
    }
  },
  loadMe: async () => {
    const token = session.token()
    if (!token) {
      set({ user: null, token: null, initialized: true })
      return
    }
    try {
      const response = await authApi.me()
      set({ user: response.data, token, initialized: true })
    } catch {
      session.clear()
      set({ user: null, token: null, initialized: true })
    }
  },
  logout: () => {
    session.clear()
    set({ user: null, token: null, initialized: true })
  }
}))

