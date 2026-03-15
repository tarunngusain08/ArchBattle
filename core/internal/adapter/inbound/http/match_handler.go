package http

import (
	"context"
	"fmt"
	stdhttp "net/http"
	"time"

	domainauth "github.com/radhakrishna/archbattle/core/internal/domain/auth"
	domainmatchmaking "github.com/radhakrishna/archbattle/core/internal/domain/matchmaking"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type MatchRateLimiter interface {
	Allow(ctx context.Context, key string, limit int, ttl time.Duration) (bool, int64, error)
}

type MatchHandler struct {
	auth              *domainauth.Service
	matchmaking       *domainmatchmaking.Service
	rateLimiter       MatchRateLimiter
	matchLimitPerDay  int
}

type queueRequest struct {
	Tier  string `json:"tier"`
	Topic string `json:"topic"`
	Mode  string `json:"mode"`
}

func NewMatchHandler(auth *domainauth.Service, matchmaking *domainmatchmaking.Service, rateLimiter MatchRateLimiter, matchLimitPerDay int) *MatchHandler {
	return &MatchHandler{auth: auth, matchmaking: matchmaking, rateLimiter: rateLimiter, matchLimitPerDay: matchLimitPerDay}
}

func (h *MatchHandler) Queue(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	userID, ok := CurrentUserID(r.Context())
	if !ok {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "missing user context"})
		return
	}
	user, err := h.auth.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var req queueRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}

	tier := user.Tier
	if req.Tier != "" {
		parsed, err := shared.ParseTier(req.Tier)
		if err != nil {
			writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		tier = parsed
	}

	key := fmt.Sprintf("ratelimit:match:%s:%s", userID.String(), time.Now().UTC().Format("2006-01-02"))
	allowed, count, err := h.rateLimiter.Allow(r.Context(), key, h.matchLimitPerDay, 24*time.Hour)
	if err != nil {
		allowed = true // Fail open — do not block the player when Redis is down
	}
	if !allowed {
		writeJSON(w, stdhttp.StatusTooManyRequests, map[string]any{
			"error":       "daily match limit reached",
			"limit":       h.matchLimitPerDay,
			"used":        count,
			"upgrade_url": "/pricing",
		})
		return
	}

	entry := domainmatchmaking.QueueEntry{UserID: user.ID, Username: user.Username, Tier: tier, Topic: shared.NormalizeTopic(req.Topic), Mode: shared.Mode(req.Mode), ELO: user.CurrentELO(tier)}
	if err := h.matchmaking.Enqueue(r.Context(), entry); err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusAccepted, map[string]any{"status": "queued", "entry": entry})
}

func (h *MatchHandler) Dequeue(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	userID, ok := CurrentUserID(r.Context())
	if !ok {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "missing user context"})
		return
	}
	user, err := h.auth.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "topic is required"})
		return
	}
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = string(shared.ModeFFF)
	}

	entry := domainmatchmaking.QueueEntry{UserID: user.ID, Username: user.Username, Tier: user.Tier, Topic: shared.NormalizeTopic(topic), Mode: shared.Mode(mode), ELO: user.CurrentELO(user.Tier)}
	if err := h.matchmaking.Dequeue(r.Context(), entry); err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("dequeue: %v", err)})
		return
	}
	w.WriteHeader(stdhttp.StatusNoContent)
}
