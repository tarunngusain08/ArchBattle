import type { Session, User } from '../types'
import { apiFetch } from './client'

interface AuthResponse {
  user: User
  session: Session
}

export async function register(username: string, email: string, password: string) {
  return apiFetch<AuthResponse>('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ username, email, password }),
  })
}

export async function login(email: string, password: string) {
  return apiFetch<AuthResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

export async function logout() {
  await apiFetch<void>('/auth/logout', { method: 'POST' })
}
