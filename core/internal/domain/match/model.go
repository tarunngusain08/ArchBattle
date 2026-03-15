package match

import (
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type MatchState string

const (
	StateCreated     MatchState = "created"
	StateLobby       MatchState = "lobby"
	StateActive      MatchState = "active"
	StateRevealing   MatchState = "revealing"
	StateLeaderboard MatchState = "leaderboard"
	StateScoring     MatchState = "scoring"
	StateEnded       MatchState = "ended"
	StateAbandoned   MatchState = "abandoned"
)

type Match struct {
	ID          uuid.UUID    `json:"id"`
	Mode        shared.Mode  `json:"mode"`
	Topic       shared.Topic `json:"topic"`
	Tier        shared.Tier  `json:"tier"`
	Status      MatchState   `json:"status"`
	QuestionIDs []uuid.UUID  `json:"questionIds"`
	StartedAt   *time.Time   `json:"startedAt,omitempty"`
	EndedAt     *time.Time   `json:"endedAt,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
}

type MatchPlayer struct {
	MatchID      uuid.UUID        `json:"matchId"`
	UserID       uuid.UUID        `json:"userId"`
	Username     string           `json:"username"`
	Score        int              `json:"score"`
	Answers      map[string][]int `json:"answers"`
	ELOBefore    int              `json:"eloBefore"`
	ELOAfter     int              `json:"eloAfter"`
	ELODelta     int              `json:"eloDelta"`
	Disconnected bool             `json:"disconnected"`
	JoinedAt     time.Time        `json:"joinedAt"`
}

type MatchStateData struct {
	MatchID           uuid.UUID    `json:"matchId"`
	State             MatchState   `json:"state"`
	Mode              shared.Mode  `json:"mode"`
	Topic             shared.Topic `json:"topic"`
	Tier              shared.Tier  `json:"tier"`
	PlayerIDs         []uuid.UUID  `json:"playerIds"`
	QuestionIndex     int          `json:"questionIndex"`
	CurrentQuestionID uuid.UUID    `json:"currentQuestionId"`
	UpdatedAt         time.Time    `json:"updatedAt"`
}

type AnswerSubmission struct {
	ID               uuid.UUID `json:"id"`
	MatchID          uuid.UUID `json:"matchId"`
	UserID           uuid.UUID `json:"userId"`
	QuestionID       uuid.UUID `json:"questionId"`
	ChosenOptions    []int     `json:"chosenOptions"`
	IsCorrect        bool      `json:"isCorrect"`
	PointsAwarded    int       `json:"pointsAwarded"`
	ServerReceivedAt int64     `json:"serverReceivedAt"`
	ElapsedSeconds   int       `json:"elapsedSeconds"`
}

type MatchEvent struct {
	Sequence  string         `json:"sequence,omitempty"`
	Type      string         `json:"type"`
	MatchID   uuid.UUID      `json:"matchId"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"createdAt"`
}

type CreateMatchRequest struct {
	Mode    shared.Mode     `json:"mode"`
	Topic   shared.Topic    `json:"topic"`
	Tier    shared.Tier     `json:"tier"`
	Players []PlayerProfile `json:"players"`
}

type SubmitAnswerRequest struct {
	MatchID          uuid.UUID `json:"matchId"`
	UserID           uuid.UUID `json:"userId"`
	QuestionID       uuid.UUID `json:"questionId"`
	Choices          []int     `json:"choices"`
	ServerReceivedAt int64     `json:"serverReceivedAt"`
	ElapsedSeconds   int       `json:"elapsedSeconds"`
}

type PlayerProfile struct {
	UserID        uuid.UUID `json:"userId"`
	Username      string    `json:"username"`
	CurrentELO    int       `json:"currentElo"`
	MatchesPlayed int       `json:"matchesPlayed"`
}

type PlayerStanding struct {
	UserID        uuid.UUID `json:"userId"`
	Username      string    `json:"username"`
	Score         int       `json:"score"`
	ELOBefore     int       `json:"eloBefore"`
	ELOAfter      int       `json:"eloAfter"`
	ELODelta      int       `json:"eloDelta"`
	MatchesPlayed int       `json:"matchesPlayed"`
	Disconnected  bool      `json:"disconnected"`
}

type RecordChoiceRequest struct {
	MatchID          uuid.UUID `json:"matchId"`
	UserID           uuid.UUID `json:"userId"`
	QuestionID       uuid.UUID `json:"questionId"`
	Choices          []int     `json:"choices"`
	ServerReceivedAt int64     `json:"serverReceivedAt"`
}

type RoundResult struct {
	UserID        uuid.UUID `json:"userId"`
	Username      string    `json:"username"`
	PointsAwarded int       `json:"pointsAwarded"`
	IsCorrect     bool      `json:"isCorrect"`
}

type LearningSummaryRequest struct {
	MatchID   uuid.UUID        `json:"matchId"`
	Topic     shared.Topic     `json:"topic"`
	Tier      shared.Tier      `json:"tier"`
	Standings []PlayerStanding `json:"standings"`
}

type LearningSummary struct {
	Strength       string `json:"strength"`
	Weakness       string `json:"weakness"`
	Recommendation string `json:"recommendation"`
	ELONarrative   string `json:"eloNarrative"`
}
