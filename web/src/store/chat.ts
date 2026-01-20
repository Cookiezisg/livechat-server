import { create } from 'zustand'
import { api, Message, Room, User, wsUrl } from '../lib/api'

type Presence = Record<string, { online: boolean; lastSeen: string }>

type ChatState = {
  selfId: string | null
  rooms: Room[]
  users: User[]
  messages: Record<string, Message[]>
  activeRoomId: string | null
  activeDMUserId: string | null
  presence: Presence
  typing: Record<string, string[]>
  socket: WebSocket | null
  connected: boolean
  loadRooms: (token: string) => Promise<void>
  setActiveRoom: (roomId: string) => void
  setActiveDM: (userId: string | null) => void
  connect: (token: string, selfId: string) => void
  sendMessage: (payload: { roomId?: string; recipientId?: string; body: string }) => void
  sendTyping: (payload: { roomId?: string; recipientId?: string; isTyping: boolean }) => void
  loadHistory: (token: string, roomId?: string, recipientId?: string) => Promise<void>
  searchUsers: (token: string, query: string) => Promise<User[]>
}

export const useChatStore = create<ChatState>((set, get) => ({
  rooms: [],
  users: [],
  messages: {},
  activeRoomId: null,
  activeDMUserId: null,
  presence: {},
  typing: {},
  socket: null,
  connected: false,
  selfId: null,
  loadRooms: async (token) => {
    const rooms = await api.rooms(token)
    set({ rooms })
  },
  setActiveRoom: (roomId) => {
    set({ activeRoomId: roomId, activeDMUserId: null })
  },
  setActiveDM: (userId) => {
    set({ activeDMUserId: userId, activeRoomId: null })
  },
  connect: (token, selfId) => {
    const socket = new WebSocket(wsUrl(token))
    socket.onopen = () => set({ connected: true })
    socket.onclose = () => set({ connected: false, socket: null })
    socket.onmessage = (event) => {
      const payload = JSON.parse(event.data)
      if (payload.type === 'message') {
        const msg = payload.data as Message
        if (msg.roomId) {
          set((state) => ({
            messages: {
              ...state.messages,
              [msg.roomId]: [...(state.messages[msg.roomId] || []), msg],
            },
          }))
          return
        }
        const otherId = msg.senderId === selfId ? msg.recipientId : msg.senderId
        if (!otherId) return
        const key = `dm:${otherId}`
        if (!key) return
        set((state) => ({
          messages: {
            ...state.messages,
            [key]: [...(state.messages[key] || []), msg],
          },
        }))
      }
      if (payload.type === 'presence') {
        const { userId, online, lastSeen } = payload.data as { userId: string; online: boolean; lastSeen: string }
        set((state) => ({
          presence: { ...state.presence, [userId]: { online, lastSeen } },
        }))
      }
      if (payload.type === 'typing') {
        const { roomId, recipientId, userId, isTyping } = payload.data as {
          roomId?: string
          recipientId?: string
          userId: string
          isTyping: boolean
        }
        const key = roomId ?? (recipientId ? `dm:${userId}` : 'global')
        set((state) => {
          const current = new Set(state.typing[key] || [])
          if (isTyping) {
            current.add(userId)
          } else {
            current.delete(userId)
          }
          return { typing: { ...state.typing, [key]: Array.from(current) } }
        })
      }
    }
    set({ socket, selfId })
  },
  sendMessage: (payload) => {
    const socket = get().socket
    if (!socket || socket.readyState !== WebSocket.OPEN) return
    socket.send(JSON.stringify({ type: 'message', ...payload }))
  },
  sendTyping: (payload) => {
    const socket = get().socket
    if (!socket || socket.readyState !== WebSocket.OPEN) return
    socket.send(JSON.stringify({ type: 'typing', ...payload }))
  },
  loadHistory: async (token, roomId, recipientId) => {
    if (roomId) {
      const messages = await api.roomHistory(token, roomId)
      set((state) => ({
        messages: { ...state.messages, [roomId]: messages.reverse() },
      }))
      return
    }
    if (recipientId) {
      const messages = await api.dmHistory(token, recipientId)
      set((state) => ({
        messages: { ...state.messages, [`dm:${recipientId}`]: messages.reverse() },
      }))
    }
  },
  searchUsers: async (token, query) => {
    const users = await api.searchUsers(token, query)
    set({ users })
    return users
  },
}))
