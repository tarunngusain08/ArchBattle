import type { MatchStanding } from '../../types'

export function ScoreStrip({ standings }: { standings: MatchStanding[] }) {
  return (
    <section className="panel rounded-3xl p-4">
      <div className="flex flex-wrap gap-3">
        {standings.map((standing) => (
          <div key={standing.userId} className="rounded-2xl bg-slate-900/70 px-4 py-3 text-sm text-slate-100">
            <p className="font-semibold">{standing.username}</p>
            <p className="text-slate-300">{standing.score} pts</p>
          </div>
        ))}
      </div>
    </section>
  )
}
