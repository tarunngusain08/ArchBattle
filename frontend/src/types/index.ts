export type Tier = 'junior' | 'senior' | 'staff'
export type Scope = 'global' | 'weekly'

export interface MatchHistoryEntry {
  opponent: string
  score: number
  eloDelta: number
}

export interface TopicStat {
  topic: string
  correct: number
  total: number
  accuracy: number
}

export interface User {
  id: string
  username: string
  email: string
  tier: Tier
  juniorElo: number
  seniorElo: number
  staffElo: number
  matchesPlayed: number
  currentStreak: number
  longestStreak: number
  lastDailyDate?: string
  createdAt: string
  matchHistory?: MatchHistoryEntry[]
  topicStats?: TopicStat[]
  streakCalendar?: string[]
}

export interface Session {
  token: string
  userId: string
  username: string
  tier: Tier
  elo: number
  expiresAt: string
}

export interface QuestionSnapshot {
  id: string
  prompt: string
  options: string[]
  correctAnswers?: number[]
  rationale?: string
  topic: string
  difficultyTier: Tier
  mode: string
}

export interface MatchEventEnvelope {
  type: string
  sequence?: string
  matchId?: string
  createdAt: string
  payload?: Record<string, unknown>
}

export interface MatchStanding {
  userId: string
  username: string
  score: number
  eloBefore: number
  eloAfter: number
  eloDelta: number
  matchesPlayed: number
  disconnected: boolean
}

export interface DailyChallenge {
  id: string
  challengeDate: string
  theme: string
  aiSummary: string
  questions: QuestionSnapshot[]
}

export interface DailyResult {
  userId: string
  challengeDate: string
  score: number
  percentile: number
  streakDay: number
  shareCardText: string
  completedAt: string
}

export interface LeaderboardEntry {
  userId: string
  tier?: Tier
  scope?: Scope
  week?: string
  score: number
  rank: number
  username?: string
}
