package http

import (
	stdhttp "net/http"
	"strconv"
	"time"

	domainleaderboard "github.com/radhakrishna/archbattle/core/internal/domain/leaderboard"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type LeaderboardHandler struct {
	service *domainleaderboard.Service
}

func NewLeaderboardHandler(service *domainleaderboard.Service) *LeaderboardHandler {
	return &LeaderboardHandler{service: service}
}

func (h *LeaderboardHandler) Get(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	tier, err := shared.ParseTier(r.URL.Query().Get("tier"))
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	scope := domainleaderboard.Scope(r.URL.Query().Get("scope"))
	if scope == "" {
		scope = domainleaderboard.ScopeGlobal
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 50
	}
	entries, err := h.service.List(r.Context(), tier, scope, limit, time.Now().UTC())
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"entries": entries})
}
