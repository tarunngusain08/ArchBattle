package http

import (
	"context"
	stdhttp "net/http"

	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

// TutorClient is the port for forwarding tutor requests to the AI service.
type TutorClient interface {
	Tutor(ctx context.Context, body map[string]any) (map[string]any, error)
}

// TutorHandler proxies authenticated tutor requests from the client to the AI service.
// The Core adds user context (userID, tier) before forwarding.
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

	userID, ok := CurrentUserID(r.Context())
	if !ok {
		writeJSON(w, stdhttp.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		return
	}

	var body map[string]any
	if err := readJSON(r, &body); err != nil {
		writeJSON(w, stdhttp.StatusBadRequest, map[string]string{"error": "invalid json payload"})
		return
	}

	// Inject authenticated user context so the AI service can apply per-user rate limits.
	body["user_id"] = userID.String()
	if tier, ok := r.Context().Value(shared.TierContextKey).(shared.Tier); ok {
		body["tier"] = string(tier)
	}

	result, err := h.client.Tutor(r.Context(), body)
	if err != nil {
		writeJSON(w, stdhttp.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, stdhttp.StatusOK, result)
}
