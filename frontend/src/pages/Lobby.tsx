import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'

import { LobbyCard } from '../components/match/LobbyCard'
import { Button } from '../components/common/Button'
import { useMatch } from '../hooks/useMatch'
import { useSocketStore } from '../stores/socketStore'

export function LobbyPage() {
  const navigate = useNavigate()
  const match = useMatch()
  const send = useSocketStore((state) => state.send)

  useEffect(() => {
    if (match.status === 'active') {
      navigate('/battle')
    }
  }, [match.status, navigate])

  const countdown = match.lobbyCountdown

  return (
    <div className="space-y-6">
      <LobbyCard
        players={match.players.length > 0 ? match.players : ['Waiting for players...']}
        durationSeconds={countdown ?? 10}
        running={countdown != null && countdown > 0}
      />
      <div className="flex gap-3">
        <Button
          onClick={() => {
            if (match.matchId) {
              send('join_match', { matchId: match.matchId })
            }
          }}
        >
          Sync lobby
        </Button>
        <Button variant="secondary" onClick={() => navigate('/battle')}>Go to battle screen</Button>
      </div>
    </div>
  )
}
