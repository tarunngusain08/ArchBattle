import type { LeaderboardEntry, Scope, Tier } from '../types'
import { apiFetch } from './client'

interface LeaderboardResponse {
  entries: LeaderboardEntry[]
}

export async function fetchLeaderboard(tier: Tier, scope: Scope, limit = 10) {
  return apiFetch<LeaderboardResponse>(`/leaderboard?tier=${tier}&scope=${scope}&limit=${limit}`)
}
