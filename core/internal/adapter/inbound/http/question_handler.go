package http

import (
	stdhttp "net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	domainquestion "github.com/radhakrishna/archbattle/core/internal/domain/question"
)

type QuestionHandler struct {
	service *domainquestion.Service
}

type disputeRequest struct {
	Reason string `json:"reason"`
}

func NewQuestionHandler(service *domainquestion.Service) *QuestionHandler {
	return &QuestionHandler{service: service}
}

func (h *QuestionHandler) Dispute(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	userID, ok := CurrentUserID(r.Context())
	if !ok {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "missing user context"})
		return
	}
	questionID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid question id"})
		return
	}
	var req disputeRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	question, err := h.service.SubmitDispute(r.Context(), domainquestion.Dispute{QuestionID: questionID, UserID: userID, Reason: req.Reason, CreatedAt: time.Now().UTC()})
	if err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, question)
}
