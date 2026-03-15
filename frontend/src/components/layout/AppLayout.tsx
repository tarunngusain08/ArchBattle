import { NavLink, Outlet } from 'react-router-dom'

import { useWebSocket } from '../../hooks/useWebSocket'
import { usePlayerStore } from '../../stores/playerStore'
import { Button } from '../common/Button'

const navItems = [
  { to: '/rooms', label: 'Rooms' },
  { to: '/battle', label: 'Battle' },
  { to: '/daily', label: 'Daily' },
]

export function AppLayout() {
  const username = usePlayerStore((state) => state.username)
  const clear = usePlayerStore((state) => state.clear)
  useWebSocket()

  return (
    <div className="app-shell">
      <div className="mx-auto flex min-h-screen max-w-7xl flex-col px-4 py-6 sm:px-6 lg:px-8">
        <header className="panel mb-6 flex flex-col gap-4 rounded-3xl px-6 py-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">ArchBattle</p>
            <h1 className="mt-2 text-3xl font-semibold text-white">Room-based system-design sparring</h1>
            <p className="mt-2 max-w-2xl text-sm text-slate-300">
              Create or join rooms, battle in real-time, and reinforce concepts with the AI tutor.
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-3">
            {navItems.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                className={({ isActive }) =>
                  `rounded-full px-4 py-2 text-sm font-medium ${isActive ? 'bg-cyan-500 text-slate-950' : 'bg-slate-900 text-slate-200'}`
                }
              >
                {item.label}
              </NavLink>
            ))}
            {username ? (
              <span className="rounded-full bg-slate-800 px-4 py-2 text-sm font-medium text-cyan-300">
                {username}
              </span>
            ) : null}
            <Button variant="secondary" onClick={() => clear()}>
              Sign out
            </Button>
          </div>
        </header>

        <main className="flex-1">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
