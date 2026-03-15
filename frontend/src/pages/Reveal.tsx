import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '../components/common/Button'
import { RevealCard } from '../components/match/RevealCard'
import { TutorPanel } from '../components/match/TutorPanel'
import { useMatch } from '../hooks/useMatch'
import { usePlayerStore } from '../stores/playerStore'
import { useMatchStore } from '../stores/matchStore'

export function RevealPage() {
  const match = useMatch()
  const navigate = useNavigate()
  const status = useMatchStore((state) => state.status)
  const userId = usePlayerStore((state) => state.userId)
  const [tutorOpen, setTutorOpen] = useState(false)

  const playerChoices = userId ? match.reveal?.playerChoices?.[userId] : undefined

  // Navigate to /battle when the next question_broadcast arrives (status becomes 'active').
  useEffect(() => {
    if (status === 'active') {
      navigate('/battle')
    }
  }, [status, navigate])

  // Navigate to /results when match ends.
  useEffect(() => {
    if (status === 'ended' || status === 'abandoned') {
      navigate('/results')
    }
  }, [status, navigate])

  return (
    <div className="space-y-6">
      <RevealCard rationale={match.reveal?.rationale} correctAnswers={match.reveal?.correctAnswers} />
      <div className="flex gap-3">
        <Button variant="secondary" onClick={() => navigate('/results')}>
          View results
        </Button>
        <Button variant="secondary" onClick={() => setTutorOpen(true)}>
          Explain this
        </Button>
      </div>
      {tutorOpen && (
        <TutorPanel
          userId={userId}
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
