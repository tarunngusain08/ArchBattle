import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

import { Button } from '../components/common/Button'
import { createRoom, joinRoom } from '../api/room'
import { usePlayerStore } from '../stores/playerStore'
import { useMatchStore } from '../stores/matchStore'

export function RoomsPage() {
  const navigate = useNavigate()
  const userId = usePlayerStore((state) => state.userId)
  const username = usePlayerStore((state) => state.username) ?? 'Player'
  const setMatchId = useMatchStore((state) => state.setMatchId)
  const setRoomCode = useMatchStore((state) => state.setRoomCode)
  const setIsOwner = useMatchStore((state) => state.setIsOwner)
  const reset = useMatchStore((state) => state.reset)
  const [joinCode, setJoinCode] = useState('')
  const [creating, setCreating] = useState(false)
  const [joining, setJoining] = useState(false)
  const [error, setError] = useState<string>()

  const handleCreateRoom = async () => {
    if (!userId) return
    setError(undefined)
    setCreating(true)
    try {
      reset()
      const { roomCode, matchId } = await createRoom(userId, username)
      setMatchId(matchId)
      setRoomCode(roomCode)
      setIsOwner(true)
      navigate('/lobby')
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setCreating(false)
    }
  }

  const handleJoinRoom = async () => {
    if (!userId) return
    const code = joinCode.trim().toUpperCase()
    if (code.length !== 6) {
      setError('Enter a 6-character room code')
      return
    }
    setError(undefined)
    setJoining(true)
    try {
      reset()
      const { matchId } = await joinRoom(code, userId, username)
      setMatchId(matchId)
      setRoomCode(code)
      setIsOwner(false)
      navigate('/lobby')
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setJoining(false)
    }
  }

  return (
    <div className="mx-auto max-w-2xl space-y-6 pt-8">
      <section className="panel rounded-3xl p-6">
        <p className="text-xs uppercase tracking-[0.35em] text-cyan-300">Create or Join</p>
        <h1 className="mt-3 text-2xl font-semibold text-white">Rooms</h1>
        <p className="mt-2 text-slate-400">Create a room to share a code, or join with a friend&apos;s code</p>

        <div className="mt-8 grid gap-8 sm:grid-cols-2">
          <div className="rounded-2xl border border-slate-700 bg-slate-900/50 p-6">
            <h2 className="text-lg font-medium text-white">Create Room</h2>
            <p className="mt-2 text-sm text-slate-400">Start a new match and share the code with others</p>
            <Button
              className="mt-4"
              fullWidth
              disabled={creating}
              onClick={handleCreateRoom}
            >
              {creating ? 'Creating...' : 'Create Room'}
            </Button>
          </div>

          <div className="rounded-2xl border border-slate-700 bg-slate-900/50 p-6">
            <h2 className="text-lg font-medium text-white">Join Room</h2>
            <p className="mt-2 text-sm text-slate-400">Enter a 6-character code to join a match</p>
            <input
              className="mt-4 w-full rounded-xl border border-slate-700 bg-slate-900 px-4 py-3 text-slate-100 uppercase tracking-widest"
              placeholder="A3BX9K"
              maxLength={6}
              value={joinCode}
              onChange={(e) => setJoinCode(e.target.value.toUpperCase())}
              aria-label="Room code"
            />
            <Button
              className="mt-4"
              fullWidth
              variant="secondary"
              disabled={joining}
              onClick={handleJoinRoom}
            >
              {joining ? 'Joining...' : 'Join Room'}
            </Button>
          </div>
        </div>

        {error ? <p className="mt-6 text-sm text-rose-300">{error}</p> : null}
      </section>
    </div>
  )
}
