package http

import (
	"encoding/json"
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type RouterParams struct {
	TutorHandler     *TutorHandler
	DraftHandler     *DraftHandler
	VariationHandler *VariationHandler
	CalibrateHandler *CalibrateHandler
	SummaryHandler   *SummaryHandler
	AllowedOrigins   []string
	InternalSecret   string
}

func NewRouter(params RouterParams) stdhttp.Handler {
	router := chi.NewRouter()
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   params.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	router.Get("/health", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// All AI endpoints require an internal shared secret.
	router.Group(func(r chi.Router) {
		r.Use(InternalAuth(params.InternalSecret))
		r.Post("/ai/tutor", params.TutorHandler.Handle)
		r.Post("/ai/draft-question", params.DraftHandler.Handle)
		r.Post("/ai/draft-questions", params.DraftHandler.HandleBulk)
		r.Post("/ai/variation", params.VariationHandler.Handle)
		r.Post("/ai/calibrate-difficulty", params.CalibrateHandler.Handle)
		r.Post("/ai/learning-summary", params.SummaryHandler.Learning)
		r.Post("/ai/discussion-summary", params.SummaryHandler.Discussion)
	})

	return router
}
