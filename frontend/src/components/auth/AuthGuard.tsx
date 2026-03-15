import { Navigate, Outlet } from 'react-router-dom'

import { useAuthStore } from '../../stores/authStore'

/**
 * AuthGuard wraps protected routes and redirects unauthenticated users to /login.
 * It reads the token from the auth store (persisted in localStorage) so it works
 * across page reloads without requiring an API round-trip.
 */
export function AuthGuard() {
  const token = useAuthStore((state) => state.token)

  if (!token) {
    return <Navigate to="/login" replace />
  }

  return <Outlet />
}
