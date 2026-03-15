import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '../components/common/Button'
import { joinAsPlayer } from '../api/player'
import { usePlayerStore } from '../stores/playerStore'

export function EnterPage() {
  const navigate = useNavigate()
  const setPlayer = usePlayerStore((state) => state.setPlayer)
  const [username, setUsername] = useState('')
  const [error, setError] = useState<string>()

  return (
    <div className="mx-auto max-w-lg pt-16">
      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Welcome</p>
        <h1 className="mt-3 text-3xl font-semibold text-white">ArchBattle</h1>
        <p className="mt-2 text-slate-400">Enter your username to play</p>
        <div className="mt-6">
          <label className="block">
            <span className="sr-only">Username</span>
            <input
              id="enter-username"
              className="w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-slate-100"
              placeholder="Username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && document.getElementById('enter-play')?.click()}
              aria-label="Username"
            />
          </label>
        </div>
        {error ? <p className="mt-4 text-sm text-rose-300">{error}</p> : null}
        <Button
          id="enter-play"
          fullWidth
          className="mt-6"
          onClick={async () => {
            setError(undefined)
            const trimmed = username.trim()
            if (!trimmed) {
              setError('Username is required')
              return
            }
            try {
              const { userId, username: name } = await joinAsPlayer(trimmed)
              setPlayer(userId, name)
              navigate('/rooms')
            } catch (err) {
              setError((err as Error).message)
            }
          }}
        >
          Play
        </Button>
      </section>
    </div>
  )
}
