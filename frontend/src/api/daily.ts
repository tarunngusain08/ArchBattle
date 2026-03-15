import type { DailyChallenge, DailyResult } from '../types'
import { apiFetch } from './client'

export async function fetchDailyChallenge(date?: string) {
  const query = date ? `?date=${encodeURIComponent(date)}` : ''
  return apiFetch<DailyChallenge>(`/daily-challenge${query}`)
}

export async function submitDailyChallenge(payload: { date?: string; answers: Record<string, number[]>; totalMillis: number }) {
  return apiFetch<DailyResult>('/daily-submit', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function fetchShareCard(params: { date?: string; score: number; correct: number; total: number; percentile: number }) {
  const query = new URLSearchParams({
    ...(params.date ? { date: params.date } : {}),
    score: String(params.score),
    correct: String(params.correct),
    total: String(params.total),
    percentile: String(params.percentile),
  })
  return apiFetch<{ shareCardText: string }>(`/daily-share-card?${query.toString()}`)
}
