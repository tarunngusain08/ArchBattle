import { apiFetch } from './client'

export interface JoinResponse {
  userId: string
  username: string
}

export async function joinAsPlayer(username: string): Promise<JoinResponse> {
  return apiFetch<JoinResponse>('/join', {
    method: 'POST',
    body: JSON.stringify({ username }),
  })
}
