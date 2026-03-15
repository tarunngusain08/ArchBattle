import type { SocketMessage } from './types'

const DEFAULT_WS_URL = import.meta.env.VITE_WS_URL ?? 'ws://localhost:8080/ws'

export class ArchBattleSocketClient {
  private socket?: WebSocket
  private reconnectAttempts = 0
  private intentionalClose = false
  private lastSeq = ''
  private activeMatchId = ''
  private reconnectTimer?: ReturnType<typeof setTimeout>
  private readonly onMessage: (message: SocketMessage) => void
  private readonly onStatusChange?: (connected: boolean) => void
  private readonly userId: string
  private readonly username: string

  constructor(
    userId: string,
    username: string,
    onMessage: (message: SocketMessage) => void,
    onStatusChange?: (connected: boolean) => void,
  ) {
    this.userId = userId
    this.username = username
    this.onMessage = onMessage
    this.onStatusChange = onStatusChange
  }

  connect() {
    if (
      this.socket &&
      (this.socket.readyState === WebSocket.OPEN ||
        this.socket.readyState === WebSocket.CONNECTING)
    ) {
      return
    }
    this.intentionalClose = false
    const params = new URLSearchParams({ userId: this.userId, username: this.username })
    this.socket = new WebSocket(`${DEFAULT_WS_URL}?${params.toString()}`)
    this.socket.onopen = () => {
      this.reconnectAttempts = 0
      this.onStatusChange?.(true)
      // If we were in a match, send reconnect message to replay missed events.
      if (this.activeMatchId) {
        this.send('reconnect', { matchId: this.activeMatchId, lastSeq: this.lastSeq })
      }
    }
    this.socket.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data) as SocketMessage
        // Track the latest sequence number for reconnect replay.
        if (msg.sequence) {
          this.lastSeq = msg.sequence
        }
        // Track active match from match_found and join_match events.
        if (msg.type === 'match_found' && msg.payload?.match_id) {
          this.activeMatchId = String(msg.payload.match_id)
        }
        this.onMessage(msg)
      } catch (e) {
        console.warn('Invalid WS message', e)
      }
    }
    this.socket.onclose = () => {
      this.onStatusChange?.(false)
      if (this.intentionalClose) {
        return
      }
      const backoff = Math.min(1000 * 2 ** this.reconnectAttempts, 10000)
      this.reconnectAttempts += 1
      this.reconnectTimer = setTimeout(() => this.connect(), backoff)
    }
  }

  /** Intentionally close and prevent auto-reconnect. */
  disconnect() {
    this.intentionalClose = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = undefined
    }
    this.activeMatchId = ''
    this.lastSeq = ''
    this.socket?.close()
    this.socket = undefined
    this.onStatusChange?.(false)
  }

  send(type: string, payload?: Record<string, unknown>) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return
    }
    this.socket.send(JSON.stringify({ type, payload }))
  }

  setActiveMatch(matchId: string) {
    this.activeMatchId = matchId
  }
}
