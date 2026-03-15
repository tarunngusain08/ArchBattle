import { Navigate, Outlet } from 'react-router-dom'

import { usePlayerStore } from '../../stores/playerStore'

/**
 * PlayerGuard wraps protected routes and redirects users without a player identity to /.
 * It reads userId from the player store (persisted in localStorage).
 */
export function PlayerGuard() {
  const userId = usePlayerStore((state) => state.userId)

  if (!userId) {
    return <Navigate to="/" replace />
  }

  return <Outlet />
}
