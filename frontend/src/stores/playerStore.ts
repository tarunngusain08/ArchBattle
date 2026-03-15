import { create } from 'zustand'

const storedUserId = window.localStorage.getItem('archbattle.userId') ?? undefined
const storedUsername = window.localStorage.getItem('archbattle.username') ?? undefined

interface PlayerState {
  userId?: string
  username?: string
  setPlayer: (userId: string, username: string) => void
  clear: () => void
}

export const usePlayerStore = create<PlayerState>((set) => ({
  userId: storedUserId || undefined,
  username: storedUsername || undefined,
  setPlayer: (userId, username) => {
    window.localStorage.setItem('archbattle.userId', userId)
    window.localStorage.setItem('archbattle.username', username)
    set({ userId, username })
  },
  clear: () => {
    window.localStorage.removeItem('archbattle.userId')
    window.localStorage.removeItem('archbattle.username')
    set({ userId: undefined, username: undefined })
  },
}))
