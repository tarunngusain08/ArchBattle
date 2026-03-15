import { useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '../components/common/Button'
import { QuestionCard } from '../components/match/QuestionCard'
import { ScoreStrip } from '../components/match/ScoreStrip'
import { useMatch } from '../hooks/useMatch'
import { useSocketStore } from '../stores/socketStore'
import { useMatchStore } from '../stores/matchStore'

export function BattlePage() {
  const navigate = useNavigate()
  const match = useMatch()
  const send = useSocketStore((state) => state.send)
  const status = useMatchStore((state) => state.status)

  // Track when the current question started (server event createdAt) to compute elapsedSeconds.
  const questionStartRef = useRef<number>(0)
  const submittedRef = useRef(false)

  // Update the question start timestamp whenever a new question_broadcast arrives.
  const messages = useMatchStore((state) => state.messages)
  const lastBroadcastAt = messages.filter((m) => m.type === 'question_broadcast').at(-1)?.createdAt
  useEffect(() => {
    if (lastBroadcastAt) {
      questionStartRef.current = new Date(lastBroadcastAt).getTime()
      submittedRef.current = false
    }
  }, [lastBroadcastAt])

  // Navigate to /reveal when the server sends question_reveal (status becomes 'revealing').
  useEffect(() => {
    if (status === 'revealing') {
      navigate('/reveal')
    }
  }, [status, navigate])

  // Navigate to /results when match ends.
  useEffect(() => {
    if (status === 'ended' || status === 'abandoned') {
      navigate('/results')
    }
  }, [status, navigate])

  if (!match.question) {
    return (
      <section className="panel rounded-3xl p-6">
        <h2 className="text-2xl font-semibold text-white">Waiting for the next question</h2>
        <p className="mt-2 text-slate-300">
          {match.matchId
            ? 'The battle is starting soon. Hang tight...'
            : 'Create or join a room to start a battle.'}
        </p>
        {match.players.length > 0 ? (
          <div className="mt-5 flex flex-wrap gap-2">
            {match.players.map((player) => (
              <span key={player} className="rounded-full bg-slate-800 px-3 py-1 text-sm text-slate-200">
                {player}
              </span>
            ))}
          </div>
        ) : null}
        {!match.matchId ? (
          <div className="mt-5">
            <Button onClick={() => navigate('/rooms')}>Go to rooms</Button>
          </div>
        ) : null}
      </section>
    )
  }

  return (
    <div className="space-y-6">
      <ScoreStrip standings={match.standings} />
      <QuestionCard
        question={match.question}
        onChoose={(choice) => {
          if (submittedRef.current || !match.matchId) return
          submittedRef.current = true
          // Compute elapsed time from server question broadcast, not client clock delta.
          const startMs = questionStartRef.current || performance.timeOrigin
          const elapsedSeconds = Math.round((Date.now() - startMs) / 1000)
          send('answer_submit', {
            matchId: match.matchId,
            questionId: match.question?.id,
            choices: [choice],
            elapsedSeconds,
          })
          // Do NOT navigate here. Navigation is driven by the server's question_reveal event.
        }}
      />
    </div>
  )
}
