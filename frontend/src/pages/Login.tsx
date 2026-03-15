import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '../components/common/Button'
import { useAuth } from '../hooks/useAuth'

export function LoginPage() {
  const navigate = useNavigate()
  const auth = useAuth()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string>()

  return (
    <div className="mx-auto max-w-lg pt-16">
      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Welcome</p>
        <h1 className="mt-3 text-3xl font-semibold text-white">{mode === 'login' ? 'Sign in to ArchBattle' : 'Create your account'}</h1>
        <div className="mt-6 grid gap-4">
          {mode === 'register' ? (
            <label className="block">
              <span className="sr-only">Username</span>
              <input
                id="login-username"
                className="w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-slate-100"
                placeholder="Username"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                aria-label="Username"
              />
            </label>
          ) : null}
          <label className="block">
            <span className="sr-only">Email</span>
            <input
              id="login-email"
              className="w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-slate-100"
              placeholder="Email"
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              aria-label="Email"
            />
          </label>
          <label className="block">
            <span className="sr-only">Password</span>
            <input
              id="login-password"
              className="w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-slate-100"
              type="password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              aria-label="Password"
            />
          </label>
        </div>
        {error ? <p className="mt-4 text-sm text-rose-300">{error}</p> : null}
        <div className="mt-6 flex gap-3">
          <Button
            fullWidth
            onClick={async () => {
              setError(undefined)
              try {
                if (mode === 'login') {
                  await auth.login(email, password)
                } else {
                  await auth.register(username, email, password)
                }
                navigate('/')
              } catch (err) {
                setError((err as Error).message)
              }
            }}
          >
            {mode === 'login' ? 'Sign in' : 'Register'}
          </Button>
        </div>
        <button className="mt-4 bg-transparent text-sm text-cyan-300" onClick={() => setMode((current) => (current === 'login' ? 'register' : 'login'))}>
          {mode === 'login' ? 'Need an account? Register' : 'Already have an account? Sign in'}
        </button>
      </section>
    </div>
  )
}
