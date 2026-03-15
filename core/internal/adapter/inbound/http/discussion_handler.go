package http

import (
	stdhttp "net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	domaindiscussion "github.com/radhakrishna/archbattle/core/internal/domain/discussion"
)

type DiscussionHandler struct {
	service *domaindiscussion.Service
}

type createDiscussionRequest struct {
	QuestionNumber   int    `json:"questionNumber"`
	ReasoningText    string `json:"reasoningText"`
	AlternativeText  string `json:"alternativeText"`
	SurpriseText     string `json:"surpriseText"`
}

func NewDiscussionHandler(service *domaindiscussion.Service) *DiscussionHandler {
	return &DiscussionHandler{service: service}
}

func (h *DiscussionHandler) parseDate(r *stdhttp.Request) (time.Time, bool) {
	dateStr := chi.URLParam(r, "date")
	if dateStr == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func (h *DiscussionHandler) List(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	date, ok := h.parseDate(r)
	if !ok {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid date format (use YYYY-MM-DD)"})
		return
	}
	entries, err := h.service.List(r.Context(), date)
	if err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]any{"entries": entries})
}

func (h *DiscussionHandler) Create(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	userID, ok := CurrentUserID(r.Context())
	if !ok {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "missing user context"})
		return
	}
	date, ok := h.parseDate(r)
	if !ok {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid date format (use YYYY-MM-DD)"})
		return
	}
	var req createDiscussionRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	entry, err := h.service.Add(r.Context(), domaindiscussion.CreateRequest{
		ChallengeDate:   date,
		UserID:          userID,
		QuestionNumber:  req.QuestionNumber,
		ReasoningText:   req.ReasoningText,
		AlternativeText: req.AlternativeText,
		SurpriseText:    req.SurpriseText,
	})
	if err != nil {
		if err == domaindiscussion.ErrInvalidQuestionNumber {
			writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusCreated, entry)
}

func (h *DiscussionHandler) Upvote(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	userID, ok := CurrentUserID(r.Context())
	if !ok {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "missing user context"})
		return
	}
	entryIDStr := chi.URLParam(r, "id")
	entryID, err := uuid.Parse(entryIDStr)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid entry id"})
		return
	}
	if err := h.service.Upvote(r.Context(), entryID, userID); err != nil {
		writeJSON(w, stdhttp.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, map[string]string{"status": "ok"})
}
