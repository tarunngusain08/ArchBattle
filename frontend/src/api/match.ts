import { apiFetch } from './client'

export interface QueueResponse {
  status: string
  entry: {
    userId: string
    username: string
    tier: string
    topic: string
    mode: string
    elo: number
  }
}

export async function queueForMatch(tier: string, topic: string, mode = 'fff') {
  return apiFetch<QueueResponse>('/match/queue', {
    method: 'POST',
    body: JSON.stringify({ tier, topic, mode }),
  })
}

export async function leaveQueue(topic: string, mode = 'fff') {
  return apiFetch<void>(`/match/queue?topic=${encodeURIComponent(topic)}&mode=${encodeURIComponent(mode)}`, {
    method: 'DELETE',
  })
}
