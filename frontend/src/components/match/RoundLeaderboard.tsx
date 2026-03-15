import type { MatchStanding, RoundResult } from '../../types'
import { Timer } from '../common/Timer'

const medals = ['1st', '2nd', '3rd'] as const

function getMedalStyle(rank: number): string {
  switch (rank) {
    case 0:
      return 'bg-amber-500/20 text-amber-300 border-amber-500/40'
    case 1:
      return 'bg-slate-400/20 text-slate-300 border-slate-400/40'
    case 2:
      return 'bg-orange-600/20 text-orange-300 border-orange-600/40'
    default:
      return 'bg-slate-800/50 text-slate-400 border-slate-700'
  }
}

function getRankLabel(rank: number): string {
  if (rank < medals.length) return medals[rank]
  return `${rank + 1}th`
}

interface RoundLeaderboardProps {
  standings: MatchStanding[]
  roundResults?: RoundResult[]
  correctAnswers?: number[]
  rationale?: string
}

export function RoundLeaderboard({ standings, roundResults, correctAnswers, rationale }: RoundLeaderboardProps) {
  const sorted = [...standings].sort((a, b) => b.score - a.score)
  const roundMap = new Map<string, RoundResult>()
  if (roundResults) {
    for (const r of roundResults) {
      roundMap.set(r.userId, r)
    }
  }

  return (
    <section className="animate-in fade-in slide-in-from-bottom-4 space-y-5 duration-500">
      <div className="panel rounded-3xl p-6">
        <div className="flex items-center justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Leaderboard</p>
            <h2 className="mt-2 text-2xl font-semibold text-white">Round results</h2>
          </div>
          <Timer durationSeconds={6} />
        </div>

        <div className="mt-6 space-y-3">
          {sorted.map((standing, rank) => {
            const round = roundMap.get(standing.userId)
            const medalStyle = getMedalStyle(rank)
            return (
              <div
                key={standing.userId}
                className={`flex items-center gap-4 rounded-2xl border p-4 transition-all duration-300 ${medalStyle}`}
                style={{ animationDelay: `${rank * 100}ms` }}
              >
                <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-sm font-bold ${rank === 0 ? 'bg-amber-500 text-slate-900' : rank === 1 ? 'bg-slate-400 text-slate-900' : rank === 2 ? 'bg-orange-600 text-white' : 'bg-slate-700 text-slate-300'}`}>
                  {getRankLabel(rank)}
                </div>

                <div className="min-w-0 flex-1">
                  <p className="truncate text-base font-semibold text-white">{standing.username}</p>
                  <p className="text-sm text-slate-400">{standing.score} pts total</p>
                </div>

                {round ? (
                  <div className="flex items-center gap-3 text-right">
                    <span className={`text-sm font-semibold ${round.isCorrect ? 'text-emerald-400' : 'text-red-400'}`}>
                      {round.isCorrect ? 'Correct' : 'Wrong'}
                    </span>
                    {round.pointsAwarded > 0 ? (
                      <span className="rounded-full bg-emerald-500/20 px-3 py-1 text-sm font-bold text-emerald-300">
                        +{round.pointsAwarded}
                      </span>
                    ) : (
                      <span className="rounded-full bg-slate-700/50 px-3 py-1 text-sm font-bold text-slate-500">
                        +0
                      </span>
                    )}
                  </div>
                ) : null}
              </div>
            )
          })}
        </div>
      </div>

      {correctAnswers && correctAnswers.length > 0 ? (
        <div className="panel rounded-3xl p-6">
          <p className="text-xs uppercase tracking-[0.35em] text-emerald-400">Correct answer</p>
          <p className="mt-2 text-sm text-slate-300">
            Option{correctAnswers.length > 1 ? 's' : ''}: {correctAnswers.map((a) => a + 1).join(', ')}
          </p>
          {rationale ? <p className="mt-3 text-sm text-slate-400">{rationale}</p> : null}
        </div>
      ) : null}
    </section>
  )
}
