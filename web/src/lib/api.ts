const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080'

export type ApiError = {
  error: string
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, options)
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as ApiError
    throw new Error(body.error || 'Request failed')
  }
  return (await res.json()) as T
}

export const api = {
  async register(payload: { email: string; username: string; password: string; avatarUrl?: string }) {
    return request<{ token: string; user: User }>('/api/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
  },
  async login(payload: { email: string; password: string }) {
    return request<{ token: string; user: User }>('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
  },
  async me(token: string) {
    return request<User>('/api/auth/me', {
      headers: { Authorization: `Bearer ${token}` },
    })
  },
  async rooms(token: string) {
    return request<Room[]>('/api/rooms', {
      headers: { Authorization: `Bearer ${token}` },
    })
  },
  async createRoom(token: string, payload: { name: string; isPrivate: boolean }) {
    return request<Room>('/api/rooms', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    })
  },
  async joinRoom(token: string, roomId: string) {
    return request<{ status: string }>(`/api/rooms/${roomId}/join`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
    })
  },
  async roomHistory(token: string, roomId: string) {
    return request<Message[]>(`/api/rooms/${roomId}/messages`, {
      headers: { Authorization: `Bearer ${token}` },
    })
  },
  async dmHistory(token: string, userId: string) {
    return request<Message[]>(`/api/dms/${userId}/messages`, {
      headers: { Authorization: `Bearer ${token}` },
    })
  },
  async searchUsers(token: string, query: string) {
    return request<User[]>(`/api/users?query=${encodeURIComponent(query)}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
  },
}

export type User = {
  id: string
  email: string
  username: string
  avatarUrl?: string
  createdAt: string
}

export type Room = {
  id: string
  name: string
  isPrivate: boolean
  createdBy: string
  createdAt: string
}

export type Message = {
  id: string
  roomId?: string
  senderId: string
  senderName: string
  senderAvatar?: string
  recipientId?: string
  body: string
  attachmentUrl?: string
  messageType: string
  createdAt: string
}

export function wsUrl(token: string) {
  const base = API_BASE.replace('http', 'ws')
  return `${base}/ws?token=${encodeURIComponent(token)}`
}
