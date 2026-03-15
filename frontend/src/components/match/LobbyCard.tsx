import { Timer } from '../common/Timer'

export function LobbyCard({ players, durationSeconds = 10, running = false }: { players: string[]; durationSeconds?: number; running?: boolean }) {
  return (
    <section className="panel rounded-3xl p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-white">Match lobby</h2>
          <p className="mt-1 text-sm text-slate-300">Players are joining. Countdown starts when the room is ready.</p>
        </div>
        <Timer durationSeconds={durationSeconds} running={running} />
      </div>
      <div className="mt-6 grid gap-3 sm:grid-cols-2">
        {players.map((player) => (
          <div key={player} className="rounded-2xl border border-slate-700 bg-slate-900/70 p-4 text-slate-100">
            {player}
          </div>
        ))}
      </div>
    </section>
  )
}
