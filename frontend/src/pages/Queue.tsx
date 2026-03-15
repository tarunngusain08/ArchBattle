import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { leaveQueue, queueForMatch } from '../api/match'
import { Button } from '../components/common/Button'
import { Timer } from '../components/common/Timer'
import { useMatch } from '../hooks/useMatch'
import { useMatchStore } from '../stores/matchStore'
import { useQueueStore } from '../stores/queueStore'
import { useSocketStore } from '../stores/socketStore'

export function QueuePage() {
  const navigate = useNavigate()
  const { matchId } = useMatch()
  const queue = useQueueStore()
  const match = useMatchStore()
  const send = useSocketStore((state) => state.send)
  const [error, setError] = useState<string>()
  const [nowMs, setNowMs] = useState(() => Date.now())

  useEffect(() => {
    if (!queue.isQueued) {
      return undefined
    }
    const timer = window.setInterval(() => setNowMs(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [queue.isQueued])

  const waitSeconds = queue.queuedAt ? Math.max(0, Math.floor((nowMs - queue.queuedAt) / 1000)) : 0

  useEffect(() => {
    if (matchId) {
      send('join_match', { matchId })
      navigate('/lobby')
    }
  }, [matchId, navigate, send])

  return (
    <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Matchmaking</p>
        <h2 className="mt-3 text-2xl font-semibold text-white">Enter the queue</h2>
        <div className="mt-6 grid gap-4 sm:grid-cols-3">
          <label className="text-sm text-slate-300">
            Tier
            <select className="mt-2 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100" value={queue.tier} onChange={(event) => queue.setQueue({ topic: queue.topic, tier: event.target.value, mode: queue.mode })}>
              <option value="junior">Junior</option>
              <option value="senior">Senior</option>
              <option value="staff">Staff</option>
            </select>
          </label>
          <label className="text-sm text-slate-300">
            Topic
            <select className="mt-2 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100" value={queue.topic} onChange={(event) => queue.setQueue({ topic: event.target.value, tier: queue.tier, mode: queue.mode })}>
              <option value="caching">Caching</option>
              <option value="queues">Queues</option>
              <option value="storage">Storage</option>
              <option value="rate-limiting">Rate limiting</option>
              <option value="observability">Observability</option>
            </select>
          </label>
          <label className="text-sm text-slate-300">
            Mode
            <select className="mt-2 w-full rounded-xl border border-slate-700 bg-slate-900 px-3 py-2 text-slate-100" value={queue.mode} onChange={(event) => queue.setQueue({ topic: queue.topic, tier: queue.tier, mode: event.target.value })}>
              <option value="fff">Fastest Finger First</option>
            </select>
          </label>
        </div>
        <div className="mt-6 flex flex-wrap gap-3">
          <Button
            onClick={async () => {
              setError(undefined)
              try {
                await queueForMatch(queue.tier, queue.topic, queue.mode)
                queue.setQueue({ topic: queue.topic, tier: queue.tier, mode: queue.mode })
              } catch (err) {
                setError((err as Error).message)
              }
            }}
          >
            Find match
          </Button>
          <Button
            variant="secondary"
            onClick={async () => {
              await leaveQueue(queue.topic, queue.mode)
              queue.clearQueue()
              match.clearPrompts()
            }}
          >
            Leave queue
          </Button>
        </div>
        {error ? <p className="mt-4 text-sm text-rose-300">{error}</p> : null}
      </section>

      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Queue state</p>
        <h2 className="mt-3 text-xl font-semibold text-white">{queue.isQueued ? 'Searching for a match...' : 'Not queued yet'}</h2>
        <div className="mt-5 flex items-center gap-3 text-slate-300">
          <Timer durationSeconds={90} running={queue.isQueued} />
          <span>Elapsed: {waitSeconds}s</span>
        </div>
        <div className="mt-6 space-y-4">
          {match.crossMatchPrompt && (
            <div className="rounded-2xl border border-amber-500/50 bg-amber-500/10 p-4">
              <p className="text-sm font-medium text-amber-200">
                Cross-tier match available: {match.crossMatchPrompt.targetTier}
              </p>
              <p className="mt-1 text-xs text-slate-400">
                Accept to play in a different tier (expires in {match.crossMatchPrompt.timeoutSeconds}s)
              </p>
              <div className="mt-3 flex gap-2">
                <Button
                  onClick={() => {
                    send('cross_match_accept', { tier: match.crossMatchPrompt!.targetTier })
                    match.clearPrompts()
                  }}
                >
                  Accept
                </Button>
                <Button variant="secondary" onClick={() => match.clearPrompts()}>
                  Decline
                </Button>
              </div>
            </div>
          )}
          {match.soloFallbackOffer && (
            <div className="rounded-2xl border border-cyan-500/50 bg-cyan-500/10 p-4">
              <p className="text-sm font-medium text-cyan-200">
                Play solo (no ELO impact)
              </p>
              <p className="mt-1 text-xs text-slate-400">
                Practice alone without affecting your rating
              </p>
              <div className="mt-3 flex gap-2">
                <Button
                  onClick={() => {
                    send('accept_solo', {})
                    match.clearPrompts()
                  }}
                >
                  Accept
                </Button>
                <Button variant="secondary" onClick={() => match.clearPrompts()}>
                  Decline
                </Button>
              </div>
            </div>
          )}
          {!match.crossMatchPrompt && !match.soloFallbackOffer && (
            <div className="rounded-2xl border border-slate-700 bg-slate-900/70 p-4 text-sm text-slate-300">
              <p>Cross-match prompts can appear for staff players after 90s.</p>
              <p className="mt-2">Solo fallback will also be offered with no ELO impact.</p>
            </div>
          )}
        </div>
      </section>
    </div>
  )
}
