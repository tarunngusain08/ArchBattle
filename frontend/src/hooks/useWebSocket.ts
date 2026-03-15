import { useEffect, useRef } from 'react'

import type { MatchEventEnvelope } from '../types'
import { usePlayerStore } from '../stores/playerStore'
import { useMatchStore } from '../stores/matchStore'
import { useSocketStore } from '../stores/socketStore'
import { ArchBattleSocketClient } from '../ws/client'

export function useWebSocket() {
  const userId = usePlayerStore((state) => state.userId)
  const username = usePlayerStore((state) => state.username) ?? 'Player'
  const setConnected = useMatchStore((state) => state.setConnected)
  const applyEvent = useMatchStore((state) => state.applyEvent)
  const setSend = useSocketStore((state) => state.setSend)
  const clientRef = useRef<ArchBattleSocketClient | null>(null)

  useEffect(() => {
    if (!userId) {
      clientRef.current?.disconnect()
      clientRef.current = null
      return undefined
    }

    const client = new ArchBattleSocketClient(
      userId,
      username,
      (message: MatchEventEnvelope) => {
        applyEvent(message)
      },
      (connected) => setConnected(connected),
    )
    client.connect()
    clientRef.current = client
    setSend((type, payload) => client.send(type, payload))

    return () => {
      client.disconnect()
      clientRef.current = null
      setSend(() => undefined)
    }
  }, [applyEvent, setConnected, setSend, userId, username])

  return clientRef
}
