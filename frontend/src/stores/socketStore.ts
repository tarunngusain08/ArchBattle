import { create } from 'zustand'

interface SocketState {
  send: (type: string, payload?: Record<string, unknown>) => void
  setSend: (send: (type: string, payload?: Record<string, unknown>) => void) => void
}

export const useSocketStore = create<SocketState>((set) => ({
  send: () => undefined,
  setSend: (send) => set({ send }),
}))
