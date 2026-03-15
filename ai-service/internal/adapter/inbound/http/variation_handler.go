package http

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/variation"
)

type VariationHandler struct {
	service *variation.Service
}

func NewVariationHandler(service *variation.Service) *VariationHandler {
	return &VariationHandler{service: service}
}

func (h *VariationHandler) Handle(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req variation.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	response, err := h.service.Generate(r.Context(), req)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, response)
}
