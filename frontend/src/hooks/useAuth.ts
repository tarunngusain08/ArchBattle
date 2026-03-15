import { useMemo } from 'react'

import { login, logout, register } from '../api/auth'
import { useAuthStore } from '../stores/authStore'

export function useAuth() {
  const store = useAuthStore()

  return useMemo(
    () => ({
      ...store,
      isAuthenticated: Boolean(store.token),
      async login(email: string, password: string) {
        const response = await login(email, password)
        store.setAuth(response.user, response.session)
        return response
      },
      async register(username: string, email: string, password: string) {
        const response = await register(username, email, password)
        store.setAuth(response.user, response.session)
        return response
      },
      async logout() {
        if (store.token) {
          await logout().catch(() => undefined)
        }
        store.clearAuth()
      },
    }),
    [store],
  )
}
