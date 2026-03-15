import { NavLink, Outlet } from 'react-router-dom'

import { useAuth } from '../../hooks/useAuth'
import { useWebSocket } from '../../hooks/useWebSocket'
import { Button } from '../common/Button'

const navItems = [
  { to: '/', label: 'Home' },
  { to: '/queue', label: 'Queue' },
  { to: '/battle', label: 'Battle' },
  { to: '/daily', label: 'Daily' },
  { to: '/discussion', label: 'Discussion' },
  { to: '/profile', label: 'Profile' },
]

export function AppLayout() {
  const auth = useAuth()
  useWebSocket()

  return (
    <div className="app-shell">
      <div className="mx-auto flex min-h-screen max-w-7xl flex-col px-4 py-6 sm:px-6 lg:px-8">
        <header className="panel mb-6 flex flex-col gap-4 rounded-3xl px-6 py-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">ArchBattle</p>
            <h1 className="mt-2 text-3xl font-semibold text-white">Competitive system-design sparring</h1>
            <p className="mt-2 max-w-2xl text-sm text-slate-300">
              Match into real-time battles, review rationale, and reinforce concepts with the AI tutor.
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
            {auth.isAuthenticated ? (
              <Button variant="secondary" onClick={() => void auth.logout()}>
                Sign out
              </Button>
            ) : (
              <NavLink to="/login" className="rounded-full bg-cyan-500 px-4 py-2 text-sm font-semibold text-slate-950">
                Sign in
              </NavLink>
            )}
          </div>
        </header>

        <main className="flex-1">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
