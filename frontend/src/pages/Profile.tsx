import { useEffect } from 'react'
import { useAuthStore } from '../stores/authStore'

export function ProfilePage() {
  const user = useAuthStore((state) => state.user)
  const refreshUser = useAuthStore((state) => state.refreshUser)

  useEffect(() => {
    void refreshUser()
  }, [refreshUser])

  return (
    <div className="space-y-6">
      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Profile</p>
        <h2 className="mt-3 text-2xl font-semibold text-white">{user?.username ?? 'Guest player'}</h2>
        <div className="mt-6 grid gap-4 md:grid-cols-3">
          <div className="rounded-2xl bg-slate-900/70 p-4">
            <p className="text-sm text-slate-400">Junior ELO</p>
            <p className="mt-2 text-2xl font-semibold text-white">{user?.juniorElo ?? 0}</p>
          </div>
          <div className="rounded-2xl bg-slate-900/70 p-4">
            <p className="text-sm text-slate-400">Senior ELO</p>
            <p className="mt-2 text-2xl font-semibold text-white">{user?.seniorElo ?? 0}</p>
          </div>
          <div className="rounded-2xl bg-slate-900/70 p-4">
            <p className="text-sm text-slate-400">Staff ELO</p>
            <p className="mt-2 text-2xl font-semibold text-white">{user?.staffElo ?? 0}</p>
          </div>
        </div>
        <div className="mt-6 rounded-2xl border border-slate-700 bg-slate-900/70 p-4 text-slate-300">
          <p>Matches played: {user?.matchesPlayed ?? 0}</p>
          <p className="mt-2">Current streak: {user?.currentStreak ?? 0}</p>
          <p className="mt-2">Longest streak: {user?.longestStreak ?? 0}</p>
        </div>
      </section>

      {user?.matchHistory && user.matchHistory.length > 0 && (
        <section className="panel rounded-3xl p-6">
          <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Recent matches</p>
          <h2 className="mt-3 text-xl font-semibold text-white">Last 10 matches</h2>
          <div className="mt-4 space-y-2">
            {user.matchHistory.map((m, i) => (
              <div key={i} className="flex items-center justify-between rounded-xl border border-slate-700 bg-slate-900/70 px-4 py-3">
                <span className="text-slate-200">vs {m.opponent}</span>
                <span className="text-slate-400">{m.score} pts</span>
                <span className={m.eloDelta >= 0 ? 'text-emerald-400' : 'text-rose-400'}>
                  {m.eloDelta >= 0 ? '+' : ''}{m.eloDelta} ELO
                </span>
              </div>
            ))}
          </div>
        </section>
      )}

      {user?.topicStats && user.topicStats.length > 0 && (
        <section className="panel rounded-3xl p-6">
          <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Topic breakdown</p>
          <h2 className="mt-3 text-xl font-semibold text-white">Accuracy by topic</h2>
          <div className="mt-4 space-y-3">
            {user.topicStats.map((s) => (
              <div key={s.topic}>
                <div className="flex justify-between text-sm">
                  <span className="text-slate-300 capitalize">{s.topic}</span>
                  <span className="text-slate-400">{s.correct}/{s.total} ({s.accuracy.toFixed(0)}%)</span>
                </div>
                <div className="mt-1 h-2 overflow-hidden rounded-full bg-slate-800">
                  <div
                    className="h-full bg-cyan-500 transition-all"
                    style={{ width: `${Math.min(100, s.accuracy)}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      {user?.streakCalendar && user.streakCalendar.length > 0 && (
        <section className="panel rounded-3xl p-6">
          <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Streak calendar</p>
          <h2 className="mt-3 text-xl font-semibold text-white">Daily challenge activity</h2>
          <div className="mt-4 flex flex-wrap gap-1">
            {user.streakCalendar.slice(0, 30).map((d) => (
              <div
                key={d}
                className="h-6 w-6 rounded bg-emerald-500/80"
                title={d}
              />
            ))}
          </div>
        </section>
      )}
    </div>
  )
}
