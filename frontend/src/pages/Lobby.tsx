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
    if (match.matchId) {
      send('join_match', { matchId: match.matchId })
    }
  }, [match.matchId, send])

  useEffect(() => {
    if (match.status === 'active' || match.status === 'leaderboard') {
      navigate('/battle')
    }
  }, [match.status, navigate])

  const countdown = match.lobbyCountdown

  return (
    <div className="space-y-6">
      <LobbyCard
        roomCode={match.roomCode}
        players={match.players.length > 0 ? match.players : ['Waiting for players...']}
        topic={match.topic}
        durationSeconds={countdown ?? 10}
        running={countdown != null && countdown > 0}
      />
      <div className="flex items-center gap-3">
        {match.isOwner ? (
          <Button
            onClick={() => {
              if (match.matchId) {
                send('start_battle', { matchId: match.matchId })
              }
            }}
          >
            Start Battle
          </Button>
        ) : (
          <p className="text-sm text-slate-400">Waiting for the host to start the battle...</p>
        )}
      </div>
    </div>
  )
}
