package http

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/drafter"
)

type DraftHandler struct {
	service *drafter.Service
}

func NewDraftHandler(service *drafter.Service) *DraftHandler {
	return &DraftHandler{service: service}
}

func (h *DraftHandler) Handle(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req drafter.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	draft, err := h.service.Generate(r.Context(), req)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, draft)
}

func (h *DraftHandler) HandleBulk(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req drafter.BulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	drafts, err := h.service.GenerateBulk(r.Context(), req)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, drafts)
}
