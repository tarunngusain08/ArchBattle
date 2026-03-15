package match

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

const (
	lobbyCountdownSeconds    = 10
	questionTimerSeconds     = 60
	leaderboardDisplaySeconds = 6
	pollInterval             = 500 * time.Millisecond
	abandonedCheckInterval   = 2 * time.Second
)

// TransitionErrorRecorder records invalid match state transitions for observability.
// When nil, no recording is performed.
type TransitionErrorRecorder interface {
	RecordMatchStateTransitionError()
}

type Service struct {
	matches               MatchRepository
	submissions           SubmissionRepository
	stateStore            MatchStateStore
	answers               AnswerStore
	events                EventPublisher
	broadcaster           Broadcaster
	questions             QuestionProvider
	streamTTL             time.Duration
	transitionErrRecorder TransitionErrorRecorder
}

func NewService(matches MatchRepository, submissions SubmissionRepository, stateStore MatchStateStore, answers AnswerStore, events EventPublisher, broadcaster Broadcaster, questions QuestionProvider, streamTTL time.Duration, transitionErrRecorder TransitionErrorRecorder) *Service {
	if streamTTL <= 0 {
		streamTTL = 10 * time.Minute
	}
	return &Service{matches: matches, submissions: submissions, stateStore: stateStore, answers: answers, events: events, broadcaster: broadcaster, questions: questions, streamTTL: streamTTL, transitionErrRecorder: transitionErrRecorder}
}

func (s *Service) SetBroadcaster(broadcaster Broadcaster) {
	s.broadcaster = broadcaster
}

func (s *Service) CreateMatch(ctx context.Context, req CreateMatchRequest) (*Match, error) {
	if len(req.Players) < 1 {
		return nil, fmt.Errorf("at least one player is required")
	}

	matchID := uuid.New()
	now := time.Now().UTC()
	created := &Match{ID: matchID, Mode: req.Mode, Topic: req.Topic, Tier: req.Tier, Status: StateCreated, QuestionIDs: make([]uuid.UUID, 0, shared.QuestionsPerMatch), CreatedAt: now}
	if err := s.matches.Create(ctx, created); err != nil {
		return nil, fmt.Errorf("create match: %w", err)
	}

	rows := make([]MatchPlayer, 0, len(req.Players))
	playerIDs := make([]uuid.UUID, 0, len(req.Players))
	for _, player := range req.Players {
		rows = append(rows, MatchPlayer{MatchID: matchID, UserID: player.UserID, Username: player.Username, Answers: map[string][]int{}, ELOBefore: player.CurrentELO, JoinedAt: now})
		playerIDs = append(playerIDs, player.UserID)
	}
	if err := s.matches.AddPlayers(ctx, rows); err != nil {
		return nil, fmt.Errorf("add players: %w", err)
	}

	state := &MatchStateData{MatchID: matchID, State: StateLobby, Mode: req.Mode, Topic: req.Topic, Tier: req.Tier, PlayerIDs: playerIDs, UpdatedAt: now}
	if err := s.stateStore.SetMatchState(ctx, matchID, state); err != nil {
		return nil, fmt.Errorf("set match state: %w", err)
	}
	if err := s.stateStore.SetExpiry(ctx, matchID, s.streamTTL); err != nil {
		return nil, fmt.Errorf("set match ttl: %w", err)
	}

	// Sync Postgres match status to lobby
	if err := s.matches.UpdateStatus(ctx, matchID, StateLobby); err != nil {
		return nil, fmt.Errorf("update status to lobby: %w", err)
	}

	playerProfiles := make([]map[string]any, len(req.Players))
	for i, p := range req.Players {
		playerProfiles[i] = map[string]any{"id": p.UserID.String(), "username": p.Username}
	}
	if err := s.publishAndBroadcast(ctx, matchID, &MatchEvent{Type: "match_created", MatchID: matchID, CreatedAt: now, Payload: map[string]any{"players": playerIDs, "player_profiles": playerProfiles, "topic": req.Topic, "tier": req.Tier, "mode": req.Mode}}); err != nil {
		return nil, err
	}
	created.Status = StateLobby
	return created, nil
}

func (s *Service) AddPlayerToMatch(ctx context.Context, matchID, userID uuid.UUID, username string) error {
	state, err := s.stateStore.GetMatchState(ctx, matchID)
	if err != nil {
		return fmt.Errorf("get match state: %w", err)
	}
	if state == nil {
		return fmt.Errorf("match not found")
	}
	now := time.Now().UTC()
	player := MatchPlayer{MatchID: matchID, UserID: userID, Username: username, Answers: map[string][]int{}, ELOBefore: 1000, JoinedAt: now}
	if err := s.matches.AddPlayers(ctx, []MatchPlayer{player}); err != nil {
		return fmt.Errorf("add player to match: %w", err)
	}
	if err := s.stateStore.AppendPlayer(ctx, matchID, userID); err != nil {
		return fmt.Errorf("append player: %w", err)
	}
	return nil
}

func (s *Service) JoinMatch(ctx context.Context, matchID, userID uuid.UUID) error {
	state, err := s.stateStore.GetMatchState(ctx, matchID)
	if err != nil {
		return fmt.Errorf("get match state: %w", err)
	}
	if state == nil {
		return fmt.Errorf("match not found")
	}
	if err := s.stateStore.AppendPlayer(ctx, matchID, userID); err != nil {
		return fmt.Errorf("append player: %w", err)
	}
	updatedState, err := s.stateStore.GetMatchState(ctx, matchID)
	if err != nil {
		return fmt.Errorf("get updated state: %w", err)
	}
	playerRows, err := s.matches.GetPlayers(ctx, matchID)
	if err != nil {
		return fmt.Errorf("get players: %w", err)
	}
	playerProfiles := make([]map[string]any, len(playerRows))
	for i, p := range playerRows {
		playerProfiles[i] = map[string]any{"id": p.UserID.String(), "username": p.Username}
	}
	return s.publishAndBroadcast(ctx, matchID, &MatchEvent{Type: "lobby_state", MatchID: matchID, CreatedAt: time.Now().UTC(), Payload: map[string]any{"players": updatedState.PlayerIDs, "player_profiles": playerProfiles, "joined": userID, "count": len(updatedState.PlayerIDs)}})
}

func (s *Service) StartNextQuestion(ctx context.Context, matchID uuid.UUID, exclude []uuid.UUID, excludePilot bool) (*shared.QuestionSnapshot, error) {
	state, err := s.stateStore.GetMatchState(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("get match state: %w", err)
	}
	if state == nil {
		return nil, fmt.Errorf("match not found")
	}

	question, err := s.questions.SelectQuestion(ctx, state.PlayerIDs, state.Tier, state.Topic, state.Mode, exclude, excludePilot)
	if err != nil {
		return nil, fmt.Errorf("select question: %w", err)
	}

	matchRecord, err := s.matches.FindByID(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("load match: %w", err)
	}
	matchRecord.QuestionIDs = append(matchRecord.QuestionIDs, question.ID)
	if err := s.matches.UpdateQuestionIDs(ctx, matchID, matchRecord.QuestionIDs); err != nil {
		return nil, fmt.Errorf("persist question ids: %w", err)
	}

	state.State = StateActive
	state.QuestionIndex = len(matchRecord.QuestionIDs) - 1
	state.CurrentQuestionID = question.ID
	state.UpdatedAt = time.Now().UTC()
	if err := s.stateStore.SetMatchState(ctx, matchID, state); err != nil {
		return nil, fmt.Errorf("update live state: %w", err)
	}
	if err := s.stateStore.SetCurrentQuestion(ctx, matchID, question.ID, state.QuestionIndex); err != nil {
		return nil, fmt.Errorf("set current question: %w", err)
	}

	if err := s.publishAndBroadcast(ctx, matchID, &MatchEvent{Type: "question_broadcast", MatchID: matchID, CreatedAt: state.UpdatedAt, Payload: map[string]any{"question": question.ToClientSnapshot(), "timer_s": questionTimerSeconds, "index": state.QuestionIndex}}); err != nil {
		return nil, err
	}
	return question, nil
}

// RecordChoice stores the player's latest answer selection. Called on every
// answer_submit WS message. The choice is overwritten each time; only the final
// selection at timeout is scored.
func (s *Service) RecordChoice(ctx context.Context, req RecordChoiceRequest) error {
	if err := s.answers.StoreLatestChoice(ctx, req.MatchID, req.QuestionID, req.UserID, req.Choices); err != nil {
		return fmt.Errorf("store latest choice: %w", err)
	}
	seq, err := s.answers.IncrementSeq(ctx, req.MatchID, req.QuestionID)
	if err != nil {
		return fmt.Errorf("increment answer sequence: %w", err)
	}
	orderingScore := BuildOrderingScore(req.ServerReceivedAt, seq)
	_, _ = s.answers.RecordFirstAnswerTime(ctx, req.MatchID, req.QuestionID, req.UserID, orderingScore)
	return nil
}

// ScoreRound evaluates all final answers after the question timer expires.
// It reads the latest choices from Redis, scores them, persists submissions,
// and returns per-player round results plus cumulative standings.
func (s *Service) ScoreRound(ctx context.Context, matchID uuid.UUID, question *shared.QuestionSnapshot) ([]RoundResult, []PlayerStanding, error) {
	choices, err := s.answers.GetAllLatestChoices(ctx, matchID, question.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("get final choices: %w", err)
	}

	players, err := s.matches.GetPlayers(ctx, matchID)
	if err != nil {
		return nil, nil, fmt.Errorf("get players: %w", err)
	}

	totalAnswered, _ := s.answers.GetTotalAnswered(ctx, matchID, question.ID)

	roundResults := make([]RoundResult, 0, len(players))
	for _, player := range players {
		playerChoices, answered := choices[player.UserID]
		correct := answered && IsCorrectAnswer(playerChoices, question.CorrectAnswers)

		var rank int64
		if answered {
			r, err := s.answers.GetRank(ctx, matchID, question.ID, player.UserID)
			if err == nil {
				rank = r
			}
		}

		// elapsedSeconds=0: the late-answer bonus (>45s) does not apply in
		// deferred scoring; speed advantage is captured entirely by rank.
		points := CalculatePoints(rank, totalAnswered, correct, 0)

		submission := &AnswerSubmission{
			ID:               uuid.New(),
			MatchID:          matchID,
			UserID:           player.UserID,
			QuestionID:       question.ID,
			ChosenOptions:    append([]int(nil), playerChoices...),
			IsCorrect:        correct,
			PointsAwarded:    points,
			ServerReceivedAt: time.Now().UTC().UnixNano(),
			ElapsedSeconds:   questionTimerSeconds,
		}
		if err := s.submissions.SaveSubmission(ctx, submission); err != nil {
			return nil, nil, fmt.Errorf("save submission for %s: %w", player.UserID, err)
		}

		roundResults = append(roundResults, RoundResult{
			UserID:        player.UserID,
			Username:      player.Username,
			PointsAwarded: points,
			IsCorrect:     correct,
		})
	}

	allSubmissions, err := s.submissions.ListByMatch(ctx, matchID)
	if err != nil {
		return nil, nil, fmt.Errorf("list submissions: %w", err)
	}
	cumulativeScores := map[uuid.UUID]int{}
	for _, sub := range allSubmissions {
		cumulativeScores[sub.UserID] += sub.PointsAwarded
	}

	standings := make([]PlayerStanding, 0, len(players))
	for _, player := range players {
		standings = append(standings, PlayerStanding{
			UserID:       player.UserID,
			Username:     player.Username,
			Score:        cumulativeScores[player.UserID],
			Disconnected: player.Disconnected,
		})
	}

	return roundResults, standings, nil
}

// SubmitAnswer is kept for backward compatibility (e.g. daily challenges).
func (s *Service) SubmitAnswer(ctx context.Context, req SubmitAnswerRequest, correctAnswers []int) (*AnswerSubmission, int64, error) {
	seq, err := s.answers.IncrementSeq(ctx, req.MatchID, req.QuestionID)
	if err != nil {
		return nil, 0, fmt.Errorf("increment answer sequence: %w", err)
	}
	orderingScore := BuildOrderingScore(req.ServerReceivedAt, seq)
	accepted, err := s.answers.RecordAnswer(ctx, req.MatchID, req.QuestionID, req.UserID, orderingScore)
	if err != nil {
		return nil, 0, fmt.Errorf("record answer: %w", err)
	}
	if !accepted {
		return nil, seq, fmt.Errorf("duplicate submission ignored")
	}

	rank, err := s.answers.GetRank(ctx, req.MatchID, req.QuestionID, req.UserID)
	if err != nil {
		return nil, 0, fmt.Errorf("get answer rank: %w", err)
	}
	totalAfter, err := s.answers.GetTotalAnswered(ctx, req.MatchID, req.QuestionID)
	if err != nil {
		return nil, 0, fmt.Errorf("get total answered: %w", err)
	}

	correct := IsCorrectAnswer(req.Choices, correctAnswers)
	submission := &AnswerSubmission{ID: uuid.New(), MatchID: req.MatchID, UserID: req.UserID, QuestionID: req.QuestionID, ChosenOptions: append([]int(nil), req.Choices...), IsCorrect: correct, PointsAwarded: CalculatePoints(rank, totalAfter, correct, req.ElapsedSeconds), ServerReceivedAt: req.ServerReceivedAt, ElapsedSeconds: req.ElapsedSeconds}
	if err := s.submissions.SaveSubmission(ctx, submission); err != nil {
		return nil, 0, fmt.Errorf("save submission: %w", err)
	}

	if err := s.publishAndBroadcast(ctx, req.MatchID, &MatchEvent{Type: "score_update", MatchID: req.MatchID, CreatedAt: time.Now().UTC(), Payload: map[string]any{"user_id": req.UserID, "question_id": req.QuestionID, "points_awarded": submission.PointsAwarded, "is_correct": correct}}); err != nil {
		return nil, 0, err
	}

	return submission, seq, nil
}

func (s *Service) RevealQuestion(ctx context.Context, matchID uuid.UUID, question *shared.QuestionSnapshot, choices map[string][]int) error {
	state, err := s.stateStore.GetMatchState(ctx, matchID)
	if err != nil {
		return fmt.Errorf("get match state: %w", err)
	}
	if state == nil {
		return fmt.Errorf("match not found")
	}
	if err := MustTransition(state.State, StateRevealing); err != nil {
		if s.transitionErrRecorder != nil {
			s.transitionErrRecorder.RecordMatchStateTransitionError()
		}
		return err
	}

	state.State = StateRevealing
	state.UpdatedAt = time.Now().UTC()
	if err := s.stateStore.SetMatchState(ctx, matchID, state); err != nil {
		return fmt.Errorf("set reveal state: %w", err)
	}

	return s.publishAndBroadcast(ctx, matchID, &MatchEvent{Type: "question_reveal", MatchID: matchID, CreatedAt: state.UpdatedAt, Payload: map[string]any{"question_id": question.ID, "correct_answers": question.CorrectAnswers, "rationale": question.Rationale, "player_choices": choices, "timer_s": 10}})
}

func (s *Service) CompleteMatch(ctx context.Context, matchID uuid.UUID) ([]PlayerStanding, *LearningSummary, error) {
	matchRecord, err := s.matches.FindByID(ctx, matchID)
	if err != nil {
		return nil, nil, fmt.Errorf("load match: %w", err)
	}
	if matchRecord == nil {
		return nil, nil, fmt.Errorf("match not found")
	}

	players, err := s.matches.GetPlayers(ctx, matchID)
	if err != nil {
		return nil, nil, fmt.Errorf("get players: %w", err)
	}
	submissions, err := s.submissions.ListByMatch(ctx, matchID)
	if err != nil {
		return nil, nil, fmt.Errorf("list submissions: %w", err)
	}

	scores := map[uuid.UUID]int{}
	for _, submission := range submissions {
		scores[submission.UserID] += submission.PointsAwarded
	}

	standings := make([]PlayerStanding, 0, len(players))
	for _, player := range players {
		standing := PlayerStanding{
			UserID:       player.UserID,
			Username:     player.Username,
			Score:        scores[player.UserID],
			ELOBefore:    0,
			ELOAfter:     0,
			ELODelta:     0,
			MatchesPlayed: 0,
			Disconnected: player.Disconnected,
		}
		standings = append(standings, standing)

		if err := s.matches.UpdatePlayerResult(ctx, matchID, standing); err != nil {
			return nil, nil, fmt.Errorf("update match result: %w", err)
		}
	}

	now := time.Now().UTC()
	if err := s.matches.UpdateStatus(ctx, matchID, StateEnded); err != nil {
		return nil, nil, fmt.Errorf("update match status: %w", err)
	}
	if err := s.publishAndBroadcast(ctx, matchID, &MatchEvent{Type: "match_end", MatchID: matchID, CreatedAt: now, Payload: map[string]any{"standings": standings}}); err != nil {
		return nil, nil, err
	}
	for _, qID := range matchRecord.QuestionIDs {
		_ = s.answers.SetQuestionTTL(ctx, matchID, qID, 10*time.Minute)
	}

	return standings, nil, nil
}

func (s *Service) HandleDisconnect(ctx context.Context, matchID, userID uuid.UUID) error {
	if err := s.stateStore.SetPlayerStatus(ctx, matchID, userID, "disconnected"); err != nil {
		return fmt.Errorf("mark disconnected: %w", err)
	}
	return s.publishAndBroadcast(ctx, matchID, &MatchEvent{Type: "player_disconnected", MatchID: matchID, CreatedAt: time.Now().UTC(), Payload: map[string]any{"user_id": userID}})
}

func (s *Service) SetConnected(ctx context.Context, matchID, userID uuid.UUID) error {
	if err := s.stateStore.SetPlayerStatus(ctx, matchID, userID, "connected"); err != nil {
		return fmt.Errorf("mark connected: %w", err)
	}
	return s.publishAndBroadcast(ctx, matchID, &MatchEvent{Type: "player_reconnected", MatchID: matchID, CreatedAt: time.Now().UTC(), Payload: map[string]any{"user_id": userID}})
}

func (s *Service) publishAndBroadcast(ctx context.Context, matchID uuid.UUID, event *MatchEvent) error {
	// Publish to the Redis stream only. The MatchStreamReader goroutine reads
	// from the stream and fans out to connected WS clients, ensuring events are
	// delivered exactly once and only after they are persisted in the stream.
	if err := s.events.Publish(ctx, matchID, event); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}
	return nil
}

// RunMatchLoop is the server-side game orchestrator. It drives the state machine:
// LOBBY (countdown) -> ACTIVE (question + 60s timer) -> LEADERBOARD (6s) -> [repeat] -> SCORING -> ENDED
// Players can change their answer freely during the 60s window. Only the final
// selection is scored after the timer expires. A round leaderboard is displayed
// for 6 seconds between questions.
func (s *Service) RunMatchLoop(ctx context.Context, matchID uuid.UUID, expectedPlayers int) {
	logger := slog.Default().With("match_id", matchID)

	for tick := lobbyCountdownSeconds; tick >= 0; tick-- {
		select {
		case <-ctx.Done():
			s.abandonMatch(context.Background(), matchID)
			return
		default:
		}
		_ = s.publishAndBroadcast(ctx, matchID, &MatchEvent{
			Type:      "lobby_countdown",
			MatchID:   matchID,
			CreatedAt: time.Now().UTC(),
			Payload:   map[string]any{"seconds_remaining": tick},
		})
		if tick == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err := s.matches.UpdateStatus(ctx, matchID, StateActive); err != nil {
		logger.Error("update match status to active", "error", err)
	}

	state, err := s.stateStore.GetMatchState(ctx, matchID)
	if err != nil || state == nil {
		logger.Error("get match state for question fetch", "error", err)
		s.abandonMatch(context.Background(), matchID)
		return
	}

	questions, err := s.questions.SelectQuestions(ctx, shared.QuestionsPerMatch, state.PlayerIDs, state.Tier, state.Topic, state.Mode, nil, true)
	if err != nil {
		logger.Error("select questions", "error", err)
		s.abandonMatch(context.Background(), matchID)
		return
	}

	questionIDs := make([]uuid.UUID, len(questions))
	for i, q := range questions {
		questionIDs[i] = q.ID
	}
	if err := s.matches.UpdateQuestionIDs(ctx, matchID, questionIDs); err != nil {
		logger.Error("persist question ids", "error", err)
		s.abandonMatch(context.Background(), matchID)
		return
	}

	for qIdx := 0; qIdx < len(questions); qIdx++ {
		select {
		case <-ctx.Done():
			s.abandonMatch(context.Background(), matchID)
			return
		default:
		}

		if s.allPlayersDisconnected(ctx, matchID, expectedPlayers) {
			s.abandonMatch(context.Background(), matchID)
			return
		}

		question := questions[qIdx]
		loopState, loopErr := s.stateStore.GetMatchState(ctx, matchID)
		if loopErr != nil || loopState == nil {
			logger.Error("get match state", "error", loopErr)
			s.abandonMatch(context.Background(), matchID)
			return
		}
		loopState.State = StateActive
		loopState.QuestionIndex = qIdx
		loopState.CurrentQuestionID = question.ID
		loopState.UpdatedAt = time.Now().UTC()
		if err := s.stateStore.SetMatchState(ctx, matchID, loopState); err != nil {
			logger.Error("update live state", "error", err)
			s.abandonMatch(context.Background(), matchID)
			return
		}
		if err := s.stateStore.SetCurrentQuestion(ctx, matchID, question.ID, qIdx); err != nil {
			logger.Error("set current question", "error", err)
			s.abandonMatch(context.Background(), matchID)
			return
		}
		if err := s.publishAndBroadcast(ctx, matchID, &MatchEvent{
			Type:      "question_broadcast",
			MatchID:   matchID,
			CreatedAt: loopState.UpdatedAt,
			Payload:   map[string]any{"question": question.ToClientSnapshot(), "timer_s": questionTimerSeconds, "index": qIdx},
		}); err != nil {
			logger.Error("broadcast question", "error", err)
			s.abandonMatch(context.Background(), matchID)
			return
		}

		// Always wait the full 60 seconds so players can change answers.
		select {
		case <-time.After(questionTimerSeconds * time.Second):
		case <-ctx.Done():
			s.abandonMatch(context.Background(), matchID)
			return
		}

		// Score the round using final answers
		roundResults, standings, err := s.ScoreRound(ctx, matchID, question)
		if err != nil {
			logger.Error("score round", "error", err, "q_index", qIdx)
			s.abandonMatch(context.Background(), matchID)
			return
		}

		// Transition to leaderboard state
		state, err := s.stateStore.GetMatchState(ctx, matchID)
		if err == nil && state != nil {
			state.State = StateLeaderboard
			state.UpdatedAt = time.Now().UTC()
			_ = s.stateStore.SetMatchState(ctx, matchID, state)
		}

		// Build player choices map for the leaderboard payload
		finalChoices, _ := s.answers.GetAllLatestChoices(ctx, matchID, question.ID)
		playerChoicesPayload := map[string][]int{}
		for uid, c := range finalChoices {
			playerChoicesPayload[uid.String()] = c
		}

		_ = s.publishAndBroadcast(ctx, matchID, &MatchEvent{
			Type:      "round_leaderboard",
			MatchID:   matchID,
			CreatedAt: time.Now().UTC(),
			Payload: map[string]any{
				"standings":      standings,
				"round_results":  roundResults,
				"correct_answers": question.CorrectAnswers,
				"rationale":      question.Rationale,
				"question_id":    question.ID,
				"player_choices": playerChoicesPayload,
				"timer_s":        leaderboardDisplaySeconds,
			},
		})

		if question.Status == "pilot" {
			_ = s.questions.IncrementPilotAttempt(ctx, question.ID)
		}

		// Display leaderboard for 6 seconds
		select {
		case <-time.After(leaderboardDisplaySeconds * time.Second):
		case <-ctx.Done():
			s.abandonMatch(context.Background(), matchID)
			return
		}
	}

	if err := s.matches.UpdateStatus(ctx, matchID, StateScoring); err != nil {
		logger.Error("update match status to scoring", "error", err)
	}

	standings, summary, err := s.CompleteMatch(ctx, matchID)
	if err != nil {
		logger.Error("complete match", "error", err)
		return
	}

	if summary != nil {
		_ = s.publishAndBroadcast(ctx, matchID, &MatchEvent{
			Type:      "learning_summary",
			MatchID:   matchID,
			CreatedAt: time.Now().UTC(),
			Payload: map[string]any{
				"strength":       summary.Strength,
				"weakness":       summary.Weakness,
				"recommendation": summary.Recommendation,
				"elo_narrative":  summary.ELONarrative,
			},
		})
	}
	logger.Info("match completed", "standings_count", len(standings))
}

// waitForAllAnswers polls until all expectedPlayers have answered or timeout elapses.
// Returns true if all players answered before timeout.
func (s *Service) waitForAllAnswers(ctx context.Context, matchID, questionID uuid.UUID, expectedPlayers int, timeout time.Duration) bool {
	deadline := time.NewTimer(timeout)
	defer func() {
		if !deadline.Stop() {
			select {
			case <-deadline.C:
			default:
			}
		}
	}()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-deadline.C:
			return false
		case <-ticker.C:
			total, err := s.answers.GetTotalAnswered(ctx, matchID, questionID)
			if err == nil && int(total) >= expectedPlayers {
				return true
			}
		}
	}
}

// allPlayersDisconnected returns true when all players in a match are disconnected.
func (s *Service) allPlayersDisconnected(ctx context.Context, matchID uuid.UUID, expectedPlayers int) bool {
	state, err := s.stateStore.GetMatchState(ctx, matchID)
	if err != nil || state == nil {
		return false
	}
	disconnected := 0
	for _, uid := range state.PlayerIDs {
		status, err := s.stateStore.GetPlayerStatus(ctx, matchID, uid)
		if err == nil && status == "disconnected" {
			disconnected++
		}
	}
	return disconnected > 0 && disconnected >= len(state.PlayerIDs)
}

// abandonMatch transitions the match to ABANDONED state.
func (s *Service) abandonMatch(ctx context.Context, matchID uuid.UUID) {
	_ = s.matches.UpdateStatus(ctx, matchID, StateAbandoned)
	_ = s.publishAndBroadcast(ctx, matchID, &MatchEvent{
		Type:      "match_abandoned",
		MatchID:   matchID,
		CreatedAt: time.Now().UTC(),
		Payload:   map[string]any{"reason": "all_disconnected_or_timeout"},
	})
}
