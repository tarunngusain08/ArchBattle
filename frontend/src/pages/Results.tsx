import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '../components/common/Button'
import { ShareButton } from '../components/daily/ShareButton'
import { TutorPanel } from '../components/match/TutorPanel'
import { useMatch } from '../hooks/useMatch'
import { useAuthStore } from '../stores/authStore'
import { useMatchStore } from '../stores/matchStore'
import { useSocketStore } from '../stores/socketStore'

export function ResultsPage() {
  const match = useMatch()
  const navigate = useNavigate()
  const refreshUser = useAuthStore((state) => state.refreshUser)
  const reset = useMatchStore((state) => state.reset)
  const session = useAuthStore((state) => state.session)
  const send = useSocketStore((state) => state.send)
  const [tutorOpen, setTutorOpen] = useState(false)

  const playerChoices = session?.userId ? match.reveal?.playerChoices?.[session.userId] : undefined

  useEffect(() => {
    refreshUser()
  }, [refreshUser])

  return (
    <div className="space-y-6">
      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Results</p>
        <h2 className="mt-3 text-2xl font-semibold text-white">Final standings</h2>
        <div className="mt-6 grid gap-3">
          {match.standings.map((standing) => (
            <div key={standing.userId} className="rounded-2xl border border-slate-700 bg-slate-900/70 p-4">
              <div className="flex items-center justify-between text-white">
                <span className="font-semibold">{standing.username}</span>
                <span>{standing.score} pts</span>
              </div>
              <p className="mt-2 text-sm text-slate-300">
                ELO {standing.eloBefore} → {standing.eloAfter} ({standing.eloDelta >= 0 ? '+' : ''}
                {standing.eloDelta})
              </p>
            </div>
          ))}
        </div>
      </section>

      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Learning summary</p>
        <div className="mt-4">
          {match.learningSummary &&
          typeof match.learningSummary === 'object' &&
          (match.learningSummary.strength ||
            match.learningSummary.weakness ||
            match.learningSummary.recommendation ||
            match.learningSummary.elo_narrative) ? (
            <div className="space-y-4">
              {match.learningSummary.strength && (
                <div className="rounded-2xl border border-emerald-800/50 bg-emerald-950/30 p-4">
                  <p className="text-xs font-semibold uppercase tracking-wider text-emerald-400">Strength</p>
                  <p className="mt-2 text-slate-200">{String(match.learningSummary.strength)}</p>
                </div>
              )}
              {match.learningSummary.weakness && (
                <div className="rounded-2xl border border-amber-800/50 bg-amber-950/30 p-4">
                  <p className="text-xs font-semibold uppercase tracking-wider text-amber-400">Area to improve</p>
                  <p className="mt-2 text-slate-200">{String(match.learningSummary.weakness)}</p>
                </div>
              )}
              {match.learningSummary.recommendation && (
                <div className="rounded-2xl border border-cyan-800/50 bg-cyan-950/30 p-4">
                  <p className="text-xs font-semibold uppercase tracking-wider text-cyan-400">Recommendation</p>
                  <p className="mt-2 text-slate-200">{String(match.learningSummary.recommendation)}</p>
                </div>
              )}
              {match.learningSummary.elo_narrative && (
                <div className="rounded-2xl border border-slate-700 bg-slate-900/70 p-4">
                  <p className="text-xs font-semibold uppercase tracking-wider text-slate-400">ELO narrative</p>
                  <p className="mt-2 text-slate-200">{String(match.learningSummary.elo_narrative)}</p>
                </div>
              )}
            </div>
          ) : (
            <p className="text-sm text-slate-400">Generating your personalized learning summary…</p>
          )}
        </div>
        <div className="mt-5 flex gap-3">
          <Button
            onClick={() => {
              reset()
              navigate('/queue')
            }}
          >
            Queue again
          </Button>
          {match.matchId ? (
            <Button
              variant="secondary"
              onClick={() => {
                send('rematch_request', { matchId: match.matchId })
                reset()
                navigate('/queue')
              }}
            >
              Rematch
            </Button>
          ) : null}
          <Button variant="secondary" onClick={() => setTutorOpen(true)}>
            Review with AI Tutor
          </Button>
          <ShareButton
            text={
              match.learningSummary &&
              typeof match.learningSummary === 'object'
                ? [
                    match.learningSummary.strength && `Strength: ${match.learningSummary.strength}`,
                    match.learningSummary.weakness && `Area to improve: ${match.learningSummary.weakness}`,
                    match.learningSummary.recommendation && `Recommendation: ${match.learningSummary.recommendation}`,
                    match.learningSummary.elo_narrative && `ELO: ${match.learningSummary.elo_narrative}`,
                  ]
                    .filter(Boolean)
                    .join('\n')
                : ''
            }
          />
        </div>
      </section>
      {tutorOpen && (
        <TutorPanel
          questionId={match.question?.id}
          questionPrompt={match.question?.prompt}
          officialReason={match.reveal?.rationale}
          playerAnswer={playerChoices}
          onClose={() => setTutorOpen(false)}
        />
      )}
    </div>
  )
}
