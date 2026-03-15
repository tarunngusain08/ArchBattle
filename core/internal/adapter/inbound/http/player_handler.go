package http

import (
	"context"
	"strings"

	stdhttp "net/http"

	domainauth "github.com/radhakrishna/archbattle/core/internal/domain/auth"
)

type PlayerUpserter interface {
	UpsertByUsername(ctx context.Context, username string) (*domainauth.User, error)
}

type PlayerHandler struct {
	upserter PlayerUpserter
}

func NewPlayerHandler(upserter PlayerUpserter) *PlayerHandler {
	return &PlayerHandler{upserter: upserter}
}

type joinRequest struct {
	Username string `json:"username"`
}

func (h *PlayerHandler) Join(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req joinRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "username is required"})
		return
	}
	user, err := h.upserter.UpsertByUsername(r.Context(), username)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"userId": user.ID.String(), "username": user.Username})
}
