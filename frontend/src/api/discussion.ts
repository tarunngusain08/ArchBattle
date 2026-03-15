import { apiFetch } from './client'

export interface DiscussionEntry {
  id: string
  challengeDate: string
  userId: string
  username: string
  questionNumber: number
  reasoningText: string
  alternativeText: string
  surpriseText: string
  upvotes: number
  createdAt: string
}

export interface ListResponse {
  entries: DiscussionEntry[]
}

export async function listDiscussion(date: string): Promise<DiscussionEntry[]> {
  const res = await apiFetch<ListResponse>(`/daily-challenge/${date}/discussion/`)
  return res.entries
}

export interface CreateRequest {
  questionNumber: number
  reasoningText: string
  alternativeText: string
  surpriseText: string
}

export async function createDiscussionEntry(date: string, req: CreateRequest): Promise<DiscussionEntry> {
  return apiFetch<DiscussionEntry>(`/daily-challenge/${date}/discussion/`, {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

export async function upvoteEntry(date: string, entryId: string): Promise<void> {
  await apiFetch<void>(`/daily-challenge/${date}/discussion/${entryId}/upvote`, {
    method: 'POST',
  })
}
