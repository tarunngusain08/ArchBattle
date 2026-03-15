package http

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/tutor"
)

type TutorHandler struct {
	service *tutor.Service
}

func NewTutorHandler(service *tutor.Service) *TutorHandler {
	return &TutorHandler{service: service}
}

func (h *TutorHandler) Handle(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req tutor.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	res, err := h.service.Handle(r.Context(), req)
	if err != nil {
		writeJSON(w, stdhttp.StatusTooManyRequests, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, res)
}
