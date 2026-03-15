import { create } from 'zustand'

import { apiFetch } from '../api/client'
import type { Session, User } from '../types'

interface AuthState {
  user?: User
  session?: Session
  token?: string
  setAuth: (user: User, session: Session) => void
  clearAuth: () => void
  refreshUser: () => Promise<void>
}

function safeParse<T>(raw: string | null): T | undefined {
  if (!raw) return undefined
  try {
    return JSON.parse(raw) as T
  } catch {
    return undefined
  }
}

const storedToken = window.localStorage.getItem('archbattle.token') ?? undefined
const storedUser = safeParse<User>(window.localStorage.getItem('archbattle.user'))
const storedSession = safeParse<Session>(window.localStorage.getItem('archbattle.session'))

export const useAuthStore = create<AuthState>((set, get) => ({
  token: storedToken,
  user: storedUser,
  session: storedSession,
  setAuth: (user, session) => {
    window.localStorage.setItem('archbattle.token', session.token)
    window.localStorage.setItem('archbattle.user', JSON.stringify(user))
    window.localStorage.setItem('archbattle.session', JSON.stringify(session))
    set({ user, session, token: session.token })
  },
  clearAuth: () => {
    window.localStorage.removeItem('archbattle.token')
    window.localStorage.removeItem('archbattle.user')
    window.localStorage.removeItem('archbattle.session')
    set({ user: undefined, session: undefined, token: undefined })
  },
  refreshUser: async () => {
    const { token, session } = get()
    if (!token || !session) return
    try {
      const user = await apiFetch<User>('/users/me')
      if (user?.id) {
        window.localStorage.setItem('archbattle.user', JSON.stringify(user))
        set({ user })
      }
    } catch {
      // Non-critical
    }
  },
}))
