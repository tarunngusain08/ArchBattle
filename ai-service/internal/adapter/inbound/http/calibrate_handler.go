package http

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/radhakrishna/archbattle/ai-service/internal/domain/calibrator"
)

type CalibrateHandler struct {
	service *calibrator.Service
}

func NewCalibrateHandler(service *calibrator.Service) *CalibrateHandler {
	return &CalibrateHandler{service: service}
}

func (h *CalibrateHandler) Handle(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req calibrator.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}
	response, err := h.service.Calibrate(r.Context(), req)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, response)
}
