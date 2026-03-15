import { create } from 'zustand'

import type { MatchEventEnvelope, MatchStanding, QuestionSnapshot, RoundResult } from '../types'

interface PlayerScore {
  userId: string
  score: number
}

interface SoloFallbackOffer {
  noEloImpact: boolean
}

interface CrossMatchPrompt {
  targetTier: string
  timeoutSeconds: number
}

interface MatchState {
  connected: boolean
  matchId?: string
  roomCode?: string
  isOwner: boolean
  status: 'idle' | 'queued' | 'lobby' | 'active' | 'revealing' | 'leaderboard' | 'ended' | 'abandoned'
  players: string[]
  question?: QuestionSnapshot
  reveal?: { rationale?: string; correctAnswers?: number[]; playerChoices?: Record<string, number[]> }
  standings: MatchStanding[]
  scores: PlayerScore[]
  roundResults?: RoundResult[]
  learningSummary?: Record<string, unknown>
  soloFallbackOffer?: SoloFallbackOffer
  crossMatchPrompt?: CrossMatchPrompt
  disconnectedPlayers: string[]
  lobbyCountdown?: number
  messages: MatchEventEnvelope[]
  setConnected: (connected: boolean) => void
  setQueued: (matchId?: string) => void
  setMatchId: (matchId: string) => void
  setRoomCode: (roomCode: string) => void
  setIsOwner: (isOwner: boolean) => void
  applyEvent: (event: MatchEventEnvelope) => void
  clearPrompts: () => void
  reset: () => void
}

export const useMatchStore = create<MatchState>((set) => ({
  connected: false,
  isOwner: false,
  status: 'idle',
  players: [],
  standings: [],
  scores: [],
  disconnectedPlayers: [],
  messages: [],
  setConnected: (connected) => set({ connected }),
  setQueued: (matchId) => set({ status: 'queued', matchId }),
  setMatchId: (matchId) => set({ matchId }),
  setRoomCode: (roomCode) => set({ roomCode }),
  setIsOwner: (isOwner) => set({ isOwner }),
  applyEvent: (event) =>
    set((state) => {
      const nextMessages = [...state.messages, event].slice(-40)
      switch (event.type) {
        case 'match_found':
          return {
            ...state,
            matchId: String(event.payload?.match_id ?? ''),
            status: 'queued',
            crossMatchPrompt: undefined,
            soloFallbackOffer: undefined,
            messages: nextMessages,
          }
        case 'match_created':
        case 'lobby_state': {
          const profiles = event.payload?.player_profiles as Array<{ id: string; username: string }> | undefined
          const playerNames = Array.isArray(profiles) && profiles.length > 0
            ? profiles.map((p) => p.username)
            : Array.isArray(event.payload?.players)
              ? (event.payload.players as string[])
              : state.players
          return {
            ...state,
            status: 'lobby',
            players: playerNames,
            messages: nextMessages,
          }
        }
        case 'lobby_countdown':
          return {
            ...state,
            lobbyCountdown: typeof event.payload?.seconds_remaining === 'number' ? (event.payload.seconds_remaining as number) : state.lobbyCountdown,
            messages: nextMessages,
          }
        case 'question_broadcast':
          return {
            ...state,
            status: 'active',
            question: (event.payload?.question as QuestionSnapshot | undefined) ?? state.question,
            reveal: undefined,
            roundResults: undefined,
            lobbyCountdown: undefined,
            messages: nextMessages,
          }
        case 'round_leaderboard':
          return {
            ...state,
            status: 'leaderboard',
            standings: (event.payload?.standings as MatchStanding[] | undefined) ?? state.standings,
            roundResults: (event.payload?.round_results as RoundResult[] | undefined) ?? undefined,
            reveal: {
              rationale: event.payload?.rationale as string | undefined,
              correctAnswers: event.payload?.correct_answers as number[] | undefined,
              playerChoices: event.payload?.player_choices as Record<string, number[]> | undefined,
            },
            messages: nextMessages,
          }
        case 'question_reveal':
          return {
            ...state,
            status: 'revealing',
            reveal: {
              rationale: event.payload?.rationale as string | undefined,
              correctAnswers: event.payload?.correct_answers as number[] | undefined,
              playerChoices: event.payload?.player_choices as Record<string, number[]> | undefined,
            },
            messages: nextMessages,
          }
        case 'score_update': {
          const userId = event.payload?.user_id as string | undefined
          const points = event.payload?.points_awarded as number | undefined
          if (!userId || points === undefined) {
            return { ...state, messages: nextMessages }
          }
          const existing = state.scores.find((s) => s.userId === userId)
          const scores = existing
            ? state.scores.map((s) => (s.userId === userId ? { ...s, score: s.score + points } : s))
            : [...state.scores, { userId, score: points }]
          return { ...state, scores, messages: nextMessages }
        }
        case 'player_disconnected': {
          const uid = event.payload?.user_id as string | undefined
          const disconnectedPlayers = uid && !state.disconnectedPlayers.includes(uid)
            ? [...state.disconnectedPlayers, uid]
            : state.disconnectedPlayers
          return { ...state, disconnectedPlayers, messages: nextMessages }
        }
        case 'player_reconnected': {
          const uid = event.payload?.user_id as string | undefined
          const disconnectedPlayers = uid
            ? state.disconnectedPlayers.filter((id) => id !== uid)
            : state.disconnectedPlayers
          return { ...state, disconnectedPlayers, messages: nextMessages }
        }
        case 'solo_fallback_offer':
          return {
            ...state,
            soloFallbackOffer: { noEloImpact: Boolean(event.payload?.no_elo_impact) },
            messages: nextMessages,
          }
        case 'cross_match_prompt':
          return {
            ...state,
            crossMatchPrompt: {
              targetTier: String(event.payload?.target_tier ?? ''),
              timeoutSeconds: Number(event.payload?.timeout_s ?? 15),
            },
            messages: nextMessages,
          }
        case 'match_end':
          return {
            ...state,
            status: 'ended',
            standings: (event.payload?.standings as MatchStanding[] | undefined) ?? state.standings,
            messages: nextMessages,
          }
        case 'match_abandoned':
          return { ...state, status: 'abandoned', messages: nextMessages }
        case 'learning_summary':
          return { ...state, learningSummary: event.payload, messages: nextMessages }
        default:
          return { ...state, messages: nextMessages }
      }
    }),
  clearPrompts: () => set({ crossMatchPrompt: undefined, soloFallbackOffer: undefined }),
  reset: () => set({
    connected: false,
    matchId: undefined,
    roomCode: undefined,
    isOwner: false,
    status: 'idle',
    players: [],
    question: undefined,
    reveal: undefined,
    standings: [],
    scores: [],
    roundResults: undefined,
    learningSummary: undefined,
    soloFallbackOffer: undefined,
    crossMatchPrompt: undefined,
    disconnectedPlayers: [],
    lobbyCountdown: undefined,
    messages: [],
  }),
}))
