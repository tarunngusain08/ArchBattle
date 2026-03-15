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
	lobbyCountdownSeconds  = 10
	questionTimerSeconds   = 75
	revealTimerSeconds     = 10
	pollInterval           = 500 * time.Millisecond
	abandonedCheckInterval = 2 * time.Second
)

type Service struct {
	matches      MatchRepository
	submissions  SubmissionRepository
	stateStore   MatchStateStore
	answers      AnswerStore
	events       EventPublisher
	broadcaster  Broadcaster
	questions    QuestionProvider
	players      PlayerProgressStore
	leaderboards LeaderboardRecorder
	summaries    SummaryRequester
	streamTTL    time.Duration
}

func NewService(matches MatchRepository, submissions SubmissionRepository, stateStore MatchStateStore, answers AnswerStore, events EventPublisher, broadcaster Broadcaster, questions QuestionProvider, players PlayerProgressStore, leaderboards LeaderboardRecorder, summaries SummaryRequester, streamTTL time.Duration) *Service {
	if streamTTL <= 0 {
		streamTTL = 10 * time.Minute
	}
	return &Service{matches: matches, submissions: submissions, stateStore: stateStore, answers: answers, events: events, broadcaster: broadcaster, questions: questions, players: players, leaderboards: leaderboards, summaries: summaries, streamTTL: streamTTL}
}

func (s *Service) SetBroadcaster(broadcaster Broadcaster) {
	s.broadcaster = broadcaster
}

func (s *Service) SetSummaryRequester(requester SummaryRequester) {
	s.summaries = requester
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

	if err := s.publishAndBroadcast(ctx, matchID, &MatchEvent{Type: "question_broadcast", MatchID: matchID, CreatedAt: state.UpdatedAt, Payload: map[string]any{"question": question.ToClientSnapshot(), "timer_s": 75, "index": state.QuestionIndex}}); err != nil {
		return nil, err
	}
	return question, nil
}

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

	// Get actual rank from sorted set after recording to avoid race where concurrent
	// submissions both see the same totalBefore and get the same rank.
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

	ids := make([]uuid.UUID, 0, len(players))
	for _, player := range players {
		ids = append(ids, player.UserID)
	}
	profiles, err := s.players.GetPlayerProfiles(ctx, matchRecord.Tier, ids)
	if err != nil {
		return nil, nil, fmt.Errorf("get player profiles: %w", err)
	}

	profileMap := make(map[uuid.UUID]PlayerProfile, len(profiles))
	perfMap := make(map[uuid.UUID]float64, len(profiles))
	totalPerf := 0.0
	ratingTotal := 0.0
	maxPoints := float64(shared.QuestionsPerMatch * 150)
	if maxPoints == 0 {
		maxPoints = 1
	}
	for _, profile := range profiles {
		profileMap[profile.UserID] = profile
		perf := float64(scores[profile.UserID]) / maxPoints
		perfMap[profile.UserID] = perf
		totalPerf += perf
		ratingTotal += float64(profile.CurrentELO)
	}

	standings := make([]PlayerStanding, 0, len(players))
	soloMatch := len(players) == 1
	for _, player := range players {
		profile := profileMap[player.UserID]
		oppCount := len(profiles) - 1
		meanOppPerf := 0.5
		meanOppRating := float64(profile.CurrentELO)
		if oppCount > 0 {
			meanOppPerf = (totalPerf - perfMap[player.UserID]) / float64(oppCount)
			meanOppRating = (ratingTotal - float64(profile.CurrentELO)) / float64(oppCount)
		}

		delta := 0
		if !soloMatch {
			delta = CalculateDelta(RatingInput{CurrentRating: profile.CurrentELO, MatchesPlayed: profile.MatchesPlayed, PerformanceScore: perfMap[player.UserID], MeanOpponentPerf: meanOppPerf, MeanOpponentRating: meanOppRating, ApplyDisconnectTax: player.Disconnected})
		}
		standing := PlayerStanding{UserID: player.UserID, Username: profile.Username, Score: scores[player.UserID], ELOBefore: profile.CurrentELO, ELOAfter: profile.CurrentELO + delta, ELODelta: delta, MatchesPlayed: profile.MatchesPlayed + 1, Disconnected: player.Disconnected}
		standings = append(standings, standing)

		if err := s.matches.UpdatePlayerResult(ctx, matchID, standing); err != nil {
			return nil, nil, fmt.Errorf("update match result: %w", err)
		}
		if !soloMatch {
			if err := s.players.UpdatePlayerProgress(ctx, matchRecord.Tier, standing); err != nil {
				return nil, nil, fmt.Errorf("update player progress: %w", err)
			}
			if s.leaderboards != nil {
				if err := s.leaderboards.RecordELO(ctx, matchRecord.Tier, standing.UserID, standing.ELOAfter, time.Now().UTC()); err != nil {
					return nil, nil, fmt.Errorf("record leaderboard delta: %w", err)
				}
			}
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

	var summary *LearningSummary
	if s.summaries != nil {
		var err error
		summary, err = s.summaries.RequestLearningSummary(ctx, LearningSummaryRequest{MatchID: matchID, Topic: matchRecord.Topic, Tier: matchRecord.Tier, Standings: standings})
		if err != nil {
			slog.Default().With("match_id", matchID).Warn("learning summary request failed", "error", err)
		}
	}
	return standings, summary, nil
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
// LOBBY (countdown) -> ACTIVE (question + 75s timer) -> REVEALING (10s) -> [repeat] -> SCORING -> ENDED
// It should be launched as a goroutine after CreateMatch returns.
func (s *Service) RunMatchLoop(ctx context.Context, matchID uuid.UUID, expectedPlayers int) {
	logger := slog.Default().With("match_id", matchID)

	// Phase: lobby countdown - broadcast ticks until countdown reaches 0
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

	// Update Postgres status to active
	if err := s.matches.UpdateStatus(ctx, matchID, StateActive); err != nil {
		logger.Error("update match status to active", "error", err)
	}

	seenQuestions := make([]uuid.UUID, 0, shared.QuestionsPerMatch)
	pilotSelected := false

	for qIdx := 0; qIdx < shared.QuestionsPerMatch; qIdx++ {
		// Check context before starting each question
		select {
		case <-ctx.Done():
			s.abandonMatch(context.Background(), matchID)
			return
		default:
		}

		// Check if all players abandoned
		if s.allPlayersDisconnected(ctx, matchID, expectedPlayers) {
			s.abandonMatch(context.Background(), matchID)
			return
		}

		// Start the question (exclude pilot if we already have one in this match)
		question, err := s.StartNextQuestion(ctx, matchID, seenQuestions, pilotSelected)
		if err != nil {
			logger.Error("start next question", "error", err, "q_index", qIdx)
			s.abandonMatch(context.Background(), matchID)
			return
		}
		seenQuestions = append(seenQuestions, question.ID)
		if question.Status == "pilot" {
			pilotSelected = true
		}

		// Wait for all answers OR 75s timer
		answered := s.waitForAllAnswers(ctx, matchID, question.ID, expectedPlayers, questionTimerSeconds*time.Second)
		if !answered {
			logger.Info("question timer expired", "q_index", qIdx)
		}

		// Transition to REVEALING - fetch all choices for the reveal event
		choicesMap := map[string][]int{}
		// Reveal question (broadcast correct answers and rationale)
		if err := s.RevealQuestion(ctx, matchID, question, choicesMap); err != nil {
			logger.Error("reveal question", "error", err, "q_index", qIdx)
		}

		// Increment pilot attempt after submission on pilot questions
		if question.Status == "pilot" {
			_ = s.questions.IncrementPilotAttempt(ctx, question.ID)
		}

		// Wait reveal timer
		select {
		case <-time.After(revealTimerSeconds * time.Second):
		case <-ctx.Done():
			s.abandonMatch(context.Background(), matchID)
			return
		}

		// After last question, proceed to scoring instead of next question
		if qIdx == shared.QuestionsPerMatch-1 {
			break
		}
	}

	// Update status to scoring, then run CompleteMatch
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
	// match_end already broadcast inside CompleteMatch; log completion
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
