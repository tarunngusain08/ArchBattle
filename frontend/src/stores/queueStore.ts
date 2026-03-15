import { create } from 'zustand'

interface QueueState {
  topic: string
  tier: string
  mode: string
  isQueued: boolean
  queuedAt?: number
  setQueue: (payload: { topic: string; tier: string; mode: string }) => void
  clearQueue: () => void
}

export const useQueueStore = create<QueueState>((set) => ({
  topic: 'caching',
  tier: 'junior',
  mode: 'fff',
  isQueued: false,
  setQueue: (payload) => set({ ...payload, isQueued: true, queuedAt: Date.now() }),
  clearQueue: () => set((state) => ({ ...state, isQueued: false, queuedAt: undefined })),
}))
