package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	stdhttp "net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
	domainquestion "github.com/radhakrishna/archbattle/core/internal/domain/question"
	"github.com/radhakrishna/archbattle/core/internal/observability"
)

// MatchDriver is the port the WS gateway uses to drive match operations.
// Using an interface (not *domainmatch.Service) keeps the gateway decoupled from
// the concrete service implementation.
type MatchDriver interface {
	JoinMatch(ctx context.Context, matchID, userID uuid.UUID) error
	RecordChoice(ctx context.Context, req domainmatch.RecordChoiceRequest) error
	HandleDisconnect(ctx context.Context, matchID, userID uuid.UUID) error
	SetConnected(ctx context.Context, matchID, userID uuid.UUID) error
	RunMatchLoop(ctx context.Context, matchID uuid.UUID, expectedPlayers int)
}

type StreamStarter interface {
	Start(ctx context.Context, matchID uuid.UUID)
}

type QuestionLookup interface {
	GetByID(ctx context.Context, questionID uuid.UUID) (*domainquestion.Question, error)
}

type client struct {
	userID      uuid.UUID
	username    string
	conn        *websocket.Conn
	send        chan []byte
	activeMatch uuid.UUID
	messages    []time.Time
	closed      atomic.Bool
	ctx         context.Context
	cancel      context.CancelFunc
}

func (c *client) trySend(msg []byte) {
	if c.closed.Load() {
		return
	}
	select {
	case c.send <- msg:
	default:
	}
}

type MatchmakingDriver interface {
	CreateSoloMatch(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	AcceptCrossMatch(ctx context.Context, userID uuid.UUID, targetTier string) error
	RequestRematch(ctx context.Context, matchID uuid.UUID) error
}

type Gateway struct {
	matchService   MatchDriver
	matchmaking    MatchmakingDriver
	questions      QuestionLookup
	events         domainmatch.EventPublisher
	streamReader   StreamStarter
	upgrader       websocket.Upgrader
	logger         *slog.Logger
	allowedOrigins []string
	metrics        *observability.Metrics

	mu      sync.RWMutex
	clients map[uuid.UUID]*client
	matches map[uuid.UUID]map[uuid.UUID]struct{}

	// game loop management
	loopMu          sync.Mutex
	loopStarted     map[uuid.UUID]bool
	loopCancels     map[uuid.UUID]context.CancelFunc
	expectedPlayers map[uuid.UUID]int
	roomOwners      map[uuid.UUID]uuid.UUID // matchID -> ownerID
}

func NewGateway(matchService MatchDriver, matchmaking MatchmakingDriver, questions QuestionLookup, events domainmatch.EventPublisher, streamReader StreamStarter, allowedOrigins []string, logger *slog.Logger) *Gateway {
	if logger == nil {
		logger = slog.Default()
	}
	origins := allowedOrigins
	if origins == nil {
		origins = []string{}
	}
	return &Gateway{
		matchService:   matchService,
		matchmaking:    matchmaking,
		questions:     questions,
		events:        events,
		streamReader:  streamReader,
		logger:        logger,
		allowedOrigins: origins,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *stdhttp.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				for _, allowed := range origins {
					if origin == allowed {
						return true
					}
				}
				return false
			},
		},
		clients:         map[uuid.UUID]*client{},
		matches:         map[uuid.UUID]map[uuid.UUID]struct{}{},
		loopStarted:     map[uuid.UUID]bool{},
		loopCancels:     map[uuid.UUID]context.CancelFunc{},
		expectedPlayers: map[uuid.UUID]int{},
		roomOwners:      map[uuid.UUID]uuid.UUID{},
	}
}

func (g *Gateway) SetStreamReader(streamReader StreamStarter) {
	g.streamReader = streamReader
}

func (g *Gateway) SetMatchmaking(m MatchmakingDriver) {
	g.matchmaking = m
}

func (g *Gateway) SetMetrics(metrics *observability.Metrics) {
	g.metrics = metrics
}

// SetExpectedPlayers records how many players must join before the game loop starts.
// Called by the matchmaking factory after creating a match.
func (g *Gateway) SetExpectedPlayers(matchID uuid.UUID, count int) {
	g.loopMu.Lock()
	defer g.loopMu.Unlock()
	g.expectedPlayers[matchID] = count
}

// SetRoomOwner records which user created the room so only they can start the battle.
func (g *Gateway) SetRoomOwner(matchID, ownerID uuid.UUID) {
	g.loopMu.Lock()
	defer g.loopMu.Unlock()
	g.roomOwners[matchID] = ownerID
}

// CancelMatchLoop stops a running game loop (e.g., when a match is abandoned externally).
func (g *Gateway) CancelMatchLoop(matchID uuid.UUID) {
	g.loopMu.Lock()
	defer g.loopMu.Unlock()
	if cancel, ok := g.loopCancels[matchID]; ok {
		cancel()
		delete(g.loopCancels, matchID)
	}
	delete(g.loopStarted, matchID)
	delete(g.expectedPlayers, matchID)
}

func (g *Gateway) ServeHTTP(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	userIDStr := strings.TrimSpace(r.URL.Query().Get("userId"))
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	userID, err := uuid.Parse(userIDStr)
	if err != nil || userID == uuid.Nil {
		stdhttp.Error(w, "unauthorized: userId required", stdhttp.StatusUnauthorized)
		return
	}
	if username == "" {
		username = "Player"
	}

	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		g.logger.Error("upgrade websocket", "error", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	c := &client{userID: userID, username: username, conn: conn, send: make(chan []byte, 64), ctx: ctx, cancel: cancel}
	g.mu.Lock()
	g.clients[userID] = c
	g.mu.Unlock()
	if g.metrics != nil {
		g.metrics.WSConnectionsActive.Inc()
	}

	conn.SetReadLimit(4096)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	go g.writePump(c)
	go g.readPump(c)
}

func (g *Gateway) BroadcastToMatch(ctx context.Context, matchID uuid.UUID, event *domainmatch.MatchEvent) error {
	payload, err := json.Marshal(outboundEnvelope{Type: event.Type, Sequence: event.Sequence, MatchID: matchID, Payload: event.Payload, CreatedAt: event.CreatedAt})
	if err != nil {
		return err
	}
	g.mu.RLock()
	members := g.matches[matchID]
	for userID := range members {
		if c := g.clients[userID]; c != nil {
			c.trySend(payload)
		}
	}
	g.mu.RUnlock()
	return nil
}

func (g *Gateway) SendToPlayer(ctx context.Context, userID uuid.UUID, event *domainmatch.MatchEvent) error {
	payload, err := json.Marshal(outboundEnvelope{Type: event.Type, Sequence: event.Sequence, MatchID: event.MatchID, Payload: event.Payload, CreatedAt: event.CreatedAt})
	if err != nil {
		return err
	}
	g.mu.RLock()
	c := g.clients[userID]
	g.mu.RUnlock()
	if c == nil {
		return nil
	}
	c.trySend(payload)
	return nil
}

// NotifyMatchFound sends a match_found event to the player. The WS URL is not
// included because the frontend already holds an open WS connection.
func (g *Gateway) NotifyMatchFound(ctx context.Context, userID uuid.UUID, matchID uuid.UUID) error {
	return g.SendToPlayer(ctx, userID, &domainmatch.MatchEvent{Type: "match_found", MatchID: matchID, CreatedAt: time.Now().UTC(), Payload: map[string]any{"match_id": matchID}})
}

func (g *Gateway) NotifySoloFallback(ctx context.Context, userID uuid.UUID) error {
	return g.SendToPlayer(ctx, userID, &domainmatch.MatchEvent{Type: "solo_fallback_offer", CreatedAt: time.Now().UTC(), Payload: map[string]any{"no_elo_impact": true}})
}

func (g *Gateway) NotifyCrossMatch(ctx context.Context, userID uuid.UUID, targetTier string) error {
	return g.SendToPlayer(ctx, userID, &domainmatch.MatchEvent{Type: "cross_match_prompt", CreatedAt: time.Now().UTC(), Payload: map[string]any{"target_tier": targetTier, "timeout_s": 15}})
}

func (g *Gateway) attachClientToMatch(c *client, matchID uuid.UUID) {
	g.mu.Lock()
	defer g.mu.Unlock()
	c.activeMatch = matchID
	if _, ok := g.matches[matchID]; !ok {
		g.matches[matchID] = map[uuid.UUID]struct{}{}
	}
	g.matches[matchID][c.userID] = struct{}{}
}

func (g *Gateway) removeClient(c *client) {
	c.cancel()
	g.mu.Lock()
	c.closed.Store(true)
	delete(g.clients, c.userID)
	if c.activeMatch != uuid.Nil {
		if members, ok := g.matches[c.activeMatch]; ok {
			delete(members, c.userID)
			if len(members) == 0 {
				delete(g.matches, c.activeMatch)
			}
		}
	}
	g.mu.Unlock()
	if g.metrics != nil {
		g.metrics.WSConnectionsActive.Dec()
	}
	close(c.send)
	_ = c.conn.Close()
}

func (g *Gateway) allowMessage(c *client) bool {
	now := time.Now()
	cutoff := now.Add(-5 * time.Second)
	filtered := c.messages[:0]
	for _, seenAt := range c.messages {
		if seenAt.After(cutoff) {
			filtered = append(filtered, seenAt)
		}
	}
	c.messages = filtered
	if len(c.messages) >= 10 {
		return false
	}
	c.messages = append(c.messages, now)
	return true
}
