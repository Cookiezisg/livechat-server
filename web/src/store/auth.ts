import { create } from 'zustand'
import { api, User } from '../lib/api'

type AuthState = {
  token: string | null
  user: User | null
  loading: boolean
  error: string | null
  login: (email: string, password: string) => Promise<void>
  register: (email: string, username: string, password: string) => Promise<void>
  logout: () => void
  hydrate: () => Promise<void>
}

const TOKEN_KEY = 'livechat-token'

export const useAuthStore = create<AuthState>((set, get) => ({
  token: localStorage.getItem(TOKEN_KEY),
  user: null,
  loading: false,
  error: null,
  login: async (email, password) => {
    set({ loading: true, error: null })
    try {
      const result = await api.login({ email, password })
      localStorage.setItem(TOKEN_KEY, result.token)
      set({ token: result.token, user: result.user, loading: false })
    } catch (err) {
      set({ loading: false, error: (err as Error).message })
    }
  },
  register: async (email, username, password) => {
    set({ loading: true, error: null })
    try {
      const result = await api.register({ email, username, password })
      localStorage.setItem(TOKEN_KEY, result.token)
      set({ token: result.token, user: result.user, loading: false })
    } catch (err) {
      set({ loading: false, error: (err as Error).message })
    }
  },
  logout: () => {
    localStorage.removeItem(TOKEN_KEY)
    set({ token: null, user: null })
  },
  hydrate: async () => {
    const token = get().token
    if (!token) {
      return
    }
    try {
      const user = await api.me(token)
      set({ user })
    } catch {
      localStorage.removeItem(TOKEN_KEY)
      set({ token: null, user: null })
    }
  },
}))
