import { create } from 'zustand'

import type { DailyChallenge, DailyResult } from '../types'

interface DailyState {
  challenge?: DailyChallenge
  answers: Record<string, number[]>
  result?: DailyResult
  setChallenge: (challenge: DailyChallenge) => void
  setAnswer: (questionId: string, answer: number[]) => void
  setResult: (result: DailyResult) => void
  reset: () => void
}

export const useDailyStore = create<DailyState>((set) => ({
  answers: {},
  setChallenge: (challenge) => set({ challenge, answers: {}, result: undefined }),
  setAnswer: (questionId, answer) => set((state) => ({ answers: { ...state.answers, [questionId]: answer } })),
  setResult: (result) => set({ result }),
  reset: () => set({ challenge: undefined, answers: {}, result: undefined }),
}))
