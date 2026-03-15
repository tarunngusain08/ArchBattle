package http

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/summary"
)

type SummaryHandler struct {
	service *summary.Service
}

func NewSummaryHandler(service *summary.Service) *SummaryHandler {
	return &SummaryHandler{service: service}
}

func (h *SummaryHandler) Learning(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req summary.LearningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	response, err := h.service.Learning(r.Context(), req)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, response)
}

func (h *SummaryHandler) Discussion(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req summary.DiscussionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	response, err := h.service.Discussion(r.Context(), req)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, response)
}
