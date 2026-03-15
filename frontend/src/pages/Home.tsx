import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { fetchDailyChallenge } from '../api/daily'
import { fetchLeaderboard } from '../api/leaderboard'
import { DailyCard } from '../components/daily/DailyCard'
import { StreakBadge } from '../components/daily/StreakBadge'
import { Button } from '../components/common/Button'
import { useAuthStore } from '../stores/authStore'
import type { DailyChallenge, LeaderboardEntry } from '../types'

export function HomePage() {
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.user)
  const refreshUser = useAuthStore((state) => state.refreshUser)
  const [challenge, setChallenge] = useState<DailyChallenge>()
  const [leaderboard, setLeaderboard] = useState<LeaderboardEntry[]>([])

  useEffect(() => {
    void fetchDailyChallenge().then(setChallenge).catch(() => undefined)
    if (user) {
      void refreshUser()
      void fetchLeaderboard(user.tier, 'global', 5)
        .then((response) => setLeaderboard(response.entries))
        .catch(() => undefined)
    }
  }, [user, refreshUser])

  const matchHistory = user?.matchHistory ?? []

  return (
    <div className="grid gap-6 lg:grid-cols-[1.7fr_1fr]">
      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Dashboard</p>
        <h2 className="mt-3 text-3xl font-semibold text-white">Ready for the next architecture battle?</h2>
        <p className="mt-3 max-w-2xl text-slate-300">
          Queue into a live match, review the rationale, and keep your improvement loop tight with post-match AI coaching.
        </p>
        <div className="mt-6 flex flex-wrap items-center gap-3">
          <Button onClick={() => navigate('/queue')}>Quick queue</Button>
          <Button variant="secondary" onClick={() => navigate('/daily')}>Daily challenge</Button>
          {user ? <StreakBadge streak={user.currentStreak} /> : null}
        </div>
      </section>

      {user ? (
        <section className="panel rounded-3xl p-6">
          <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Your ELO</p>
          <h2 className="mt-3 text-xl font-semibold text-white">Per-tier rating</h2>
          <div className="mt-4 space-y-2">
            <div className="flex justify-between rounded-xl bg-slate-900/70 px-4 py-2">
              <span className="text-slate-300">Junior</span>
              <span className="font-semibold text-cyan-300">{user.juniorElo}</span>
            </div>
            <div className="flex justify-between rounded-xl bg-slate-900/70 px-4 py-2">
              <span className="text-slate-300">Senior</span>
              <span className="font-semibold text-cyan-300">{user.seniorElo}</span>
            </div>
            <div className="flex justify-between rounded-xl bg-slate-900/70 px-4 py-2">
              <span className="text-slate-300">Staff</span>
              <span className="font-semibold text-cyan-300">{user.staffElo}</span>
            </div>
          </div>
        </section>
      ) : null}

      <DailyCard challenge={challenge} onOpen={() => navigate('/daily')} />

      {user && (
        <section className="panel rounded-3xl p-6 lg:col-span-2">
          <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Recent matches</p>
          <h2 className="mt-3 text-xl font-semibold text-white">Last 10 matches</h2>
          {matchHistory.length > 0 ? (
            <div className="mt-4 space-y-2">
              {matchHistory.map((m, i) => (
                <div key={i} className="flex items-center justify-between rounded-xl border border-slate-700 bg-slate-900/70 px-4 py-3">
                  <span className="text-slate-200">vs {m.opponent}</span>
                  <span className="text-slate-400">{m.score} pts</span>
                  <span className={m.eloDelta >= 0 ? 'text-emerald-400' : 'text-rose-400'}>
                    {m.eloDelta >= 0 ? '+' : ''}{m.eloDelta} ELO
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="mt-4 text-slate-500">No recent matches yet.</p>
          )}
        </section>
      )}

      <section className="panel rounded-3xl p-6 lg:col-span-2">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.3em] text-cyan-300">Leaderboard</p>
            <h2 className="mt-2 text-2xl font-semibold text-white">Top players in your current tier</h2>
          </div>
          <Button variant="ghost" onClick={() => navigate('/profile')}>View profile</Button>
        </div>
        <div className="mt-5 grid gap-3 md:grid-cols-2 xl:grid-cols-5">
          {leaderboard.map((entry) => (
            <div key={entry.userId} className="rounded-2xl border border-slate-700 bg-slate-900/70 p-4">
              <p className="text-sm text-slate-400">Rank #{entry.rank}</p>
              <p className="mt-1 font-semibold text-white">{entry.username ?? entry.userId.slice(0, 8)}</p>
              <p className="mt-2 text-cyan-300">{Math.round(entry.score)} ELO</p>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}
