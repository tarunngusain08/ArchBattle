import { Timer } from '../common/Timer'

export function LobbyCard({ roomCode, players, durationSeconds = 10, running = false }: { roomCode?: string; players: string[]; durationSeconds?: number; running?: boolean }) {
  return (
    <section className="panel rounded-3xl p-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-white">Match lobby</h2>
          <p className="mt-1 text-sm text-slate-300">Players are joining. Countdown starts when the room is ready.</p>
          {roomCode ? (
            <div className="mt-3 flex items-center gap-2">
              <span className="text-xs uppercase tracking-wider text-slate-400">Room code</span>
              <code className="rounded-lg bg-cyan-950/50 px-3 py-1.5 font-mono text-lg font-bold tracking-[0.2em] text-cyan-300">
                {roomCode}
              </code>
              <button
                type="button"
                className="text-xs text-cyan-400 hover:text-cyan-300"
                onClick={() => navigator.clipboard.writeText(roomCode)}
              >
                Copy
              </button>
            </div>
          ) : null}
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
