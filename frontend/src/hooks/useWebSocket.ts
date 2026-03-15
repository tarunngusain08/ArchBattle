import { useEffect, useRef } from 'react'

import type { MatchEventEnvelope } from '../types'
import { useAuthStore } from '../stores/authStore'
import { useMatchStore } from '../stores/matchStore'
import { useSocketStore } from '../stores/socketStore'
import { ArchBattleSocketClient } from '../ws/client'

export function useWebSocket() {
  const token = useAuthStore((state) => state.token)
  const setConnected = useMatchStore((state) => state.setConnected)
  const applyEvent = useMatchStore((state) => state.applyEvent)
  const setSend = useSocketStore((state) => state.setSend)
  const clientRef = useRef<ArchBattleSocketClient | null>(null)

  useEffect(() => {
    if (!token) {
      clientRef.current?.disconnect()
      clientRef.current = null
      return undefined
    }

    const client = new ArchBattleSocketClient(
      token,
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
  }, [applyEvent, setConnected, setSend, token])

  return clientRef
}
