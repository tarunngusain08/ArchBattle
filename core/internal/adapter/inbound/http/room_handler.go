package http

import (
	"strings"

	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	domainroom "github.com/radhakrishna/archbattle/core/internal/domain/room"
)

type RoomHandler struct {
	service *domainroom.Service
}

func NewRoomHandler(service *domainroom.Service) *RoomHandler {
	return &RoomHandler{service: service}
}

type createRoomRequest struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Topic    string `json:"topic"`
}

type joinRoomRequest struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

func (h *RoomHandler) CreateRoom(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req createRoomRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	userID, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid userId"})
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = "Player"
	}
	code, matchID, chosenTopic, err := h.service.CreateRoom(r.Context(), userID, username, req.Topic)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, map[string]any{"roomCode": code, "matchId": matchID.String(), "topic": string(chosenTopic)})
}

func (h *RoomHandler) JoinRoom(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	code := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "code")))
	if len(code) != 6 {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid room code"})
		return
	}
	var req joinRoomRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	userID, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid userId"})
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		username = "Player"
	}
	matchID, err := h.service.JoinRoom(r.Context(), code, userID, username)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"matchId": matchID.String()})
}

func (h *RoomHandler) GetRoomStatus(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	code := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "code")))
	if len(code) != 6 {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid room code"})
		return
	}
	status, err := h.service.GetRoomStatus(r.Context(), code)
	if err != nil {
		writeJSON(w, stdhttp.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{
		"matchId":     status.MatchID.String(),
		"playerCount": status.PlayerCount,
		"status":      status.Status,
	})
}
