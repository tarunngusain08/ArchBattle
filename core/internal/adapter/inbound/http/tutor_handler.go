package http

import (
	"context"
	stdhttp "net/http"

	"github.com/google/uuid"
)

// TutorClient is the port for forwarding tutor requests to the AI service.
type TutorClient interface {
	Tutor(ctx context.Context, body map[string]any) (map[string]any, error)
}

// TutorHandler proxies tutor requests from the client to the AI service.
type TutorHandler struct {
	client TutorClient
}

func NewTutorHandler(client TutorClient) *TutorHandler {
	return &TutorHandler{client: client}
}

func (h *TutorHandler) Handle(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.client == nil {
		writeJSON(w, stdhttp.StatusNotImplemented, map[string]string{"error": "tutor service not configured"})
		return
	}

	var body map[string]any
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}

	if userIDStr, ok := body["userId"].(string); ok && userIDStr != "" {
		if userID, err := uuid.Parse(userIDStr); err == nil {
			body["user_id"] = userID.String()
		}
	}

	result, err := h.client.Tutor(r.Context(), body)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, result)
}
