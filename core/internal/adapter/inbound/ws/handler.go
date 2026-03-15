package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
)

type inboundEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type outboundEnvelope struct {
	Type      string         `json:"type"`
	Sequence  string         `json:"sequence,omitempty"`
	MatchID   uuid.UUID      `json:"matchId,omitempty"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

type joinPayload struct {
	MatchID string `json:"matchId"`
}

type answerPayload struct {
	MatchID        string `json:"matchId"`
	QuestionID     string `json:"questionId"`
	Choices        []int  `json:"choices"`
	ElapsedSeconds int    `json:"elapsedSeconds"`
}

type reconnectPayload struct {
	MatchID string `json:"matchId"`
	LastSeq string `json:"lastSeq"`
}

type crossMatchAcceptPayload struct {
	Tier string `json:"tier"`
}

type acceptSoloPayload struct {
	Tier  string `json:"tier"`
	Topic string `json:"topic"`
	Mode  string `json:"mode"`
}

func (g *Gateway) readPump(c *client) {
	defer func() {
		if c.activeMatch != uuid.Nil {
			_ = g.matchService.HandleDisconnect(context.Background(), c.activeMatch, c.userID)
		}
		g.removeClient(c)
	}()

	for {
		var envelope inboundEnvelope
		if err := c.conn.ReadJSON(&envelope); err != nil {
			return
		}
		if !g.allowMessage(c) {
			c.send <- mustJSON(outboundEnvelope{Type: "error", Payload: map[string]any{"message": "rate limit exceeded"}, CreatedAt: time.Now().UTC()})
			continue
		}
		start := time.Now()
		if err := g.handleMessage(c.ctx, c, envelope); err != nil {
			c.send <- mustJSON(outboundEnvelope{Type: "error", Payload: map[string]any{"message": err.Error()}, CreatedAt: time.Now().UTC()})
		}
		if g.metrics != nil {
			g.metrics.WSMessageLatencyMillis.Observe(float64(time.Since(start).Milliseconds()))
		}
	}
}

func (g *Gateway) writePump(c *client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (g *Gateway) handleMessage(ctx context.Context, c *client, envelope inboundEnvelope) error {
	switch envelope.Type {
	case "join_match":
		return g.handleJoinMatch(ctx, c, envelope.Payload)
	case "start_battle":
		return g.handleStartBattle(ctx, c, envelope.Payload)
	case "answer_submit":
		return g.handleAnswerSubmit(ctx, c, envelope.Payload)
	case "reconnect":
		return g.handleReconnect(ctx, c, envelope.Payload)
	case "cross_match_accept":
		return g.handleCrossMatchAccept(ctx, c, envelope.Payload)
	case "accept_solo":
		return g.handleAcceptSolo(ctx, c, envelope.Payload)
	case "rematch_request":
		return g.handleRematchRequest(ctx, c, envelope.Payload)
	case "ping":
		c.send <- mustJSON(outboundEnvelope{Type: "pong", CreatedAt: time.Now().UTC()})
		return nil
	default:
		return nil
	}
}

func (g *Gateway) handleJoinMatch(ctx context.Context, c *client, raw json.RawMessage) error {
	var payload joinPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	matchID, err := uuid.Parse(payload.MatchID)
	if err != nil {
		return err
	}
	g.attachClientToMatch(c, matchID)
	if g.streamReader != nil {
		g.streamReader.Start(ctx, matchID)
	}
	return g.matchService.JoinMatch(ctx, matchID, c.userID)
}

func (g *Gateway) handleStartBattle(ctx context.Context, c *client, raw json.RawMessage) error {
	var payload joinPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	matchID, err := uuid.Parse(payload.MatchID)
	if err != nil {
		return err
	}
	g.loopMu.Lock()
	defer g.loopMu.Unlock()
	owner, hasOwner := g.roomOwners[matchID]
	if hasOwner && owner != c.userID {
		return fmt.Errorf("only the room owner can start the battle")
	}
	if g.loopStarted[matchID] {
		return nil
	}
	expected := g.expectedPlayers[matchID]
	if expected <= 0 {
		expected = 2
	}
	g.loopStarted[matchID] = true
	loopCtx, cancel := context.WithCancel(context.Background())
	g.loopCancels[matchID] = cancel
	go g.matchService.RunMatchLoop(loopCtx, matchID, expected)
	return nil
}

func (g *Gateway) handleAnswerSubmit(ctx context.Context, c *client, raw json.RawMessage) error {
	start := time.Now()
	var payload answerPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	matchID, err := uuid.Parse(payload.MatchID)
	if err != nil {
		return err
	}
	questionID, err := uuid.Parse(payload.QuestionID)
	if err != nil {
		return err
	}
	question, err := g.questions.GetByID(ctx, questionID)
	if err != nil {
		return err
	}
	_, _, err = g.matchService.SubmitAnswer(ctx, domainmatch.SubmitAnswerRequest{MatchID: matchID, UserID: c.userID, QuestionID: questionID, Choices: payload.Choices, ServerReceivedAt: time.Now().UTC().UnixNano(), ElapsedSeconds: payload.ElapsedSeconds}, question.GetCorrectAnswers())
	if g.metrics != nil {
		g.metrics.AnswerSubmitLatencyMillis.Observe(float64(time.Since(start).Milliseconds()))
	}
	return err
}

func (g *Gateway) handleCrossMatchAccept(ctx context.Context, c *client, raw json.RawMessage) error {
	if g.matchmaking == nil {
		return nil
	}
	var payload crossMatchAcceptPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	tier := payload.Tier
	if tier == "" {
		tier = "senior"
	}
	if err := g.matchmaking.AcceptCrossMatch(ctx, c.userID, tier); err != nil {
		return err
	}
	c.trySend(mustJSON(outboundEnvelope{Type: "cross_match_queued", CreatedAt: time.Now().UTC()}))
	return nil
}

func (g *Gateway) handleAcceptSolo(ctx context.Context, c *client, raw json.RawMessage) error {
	if g.matchmaking == nil {
		return nil
	}
	_, err := g.matchmaking.CreateSoloMatch(ctx, c.userID)
	if err != nil {
		return err
	}
	return nil
}

func (g *Gateway) handleRematchRequest(ctx context.Context, c *client, raw json.RawMessage) error {
	if g.matchmaking == nil {
		return nil
	}
	var payload struct {
		MatchID string `json:"matchId"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || payload.MatchID == "" {
		return nil
	}
	matchID, err := uuid.Parse(payload.MatchID)
	if err != nil {
		return nil
	}
	if err := g.matchmaking.RequestRematch(ctx, matchID); err != nil {
		return err
	}
	c.trySend(mustJSON(outboundEnvelope{Type: "rematch_queued", CreatedAt: time.Now().UTC()}))
	return nil
}

func mustJSON(value any) []byte {
	payload, _ := json.Marshal(value)
	return payload
}
