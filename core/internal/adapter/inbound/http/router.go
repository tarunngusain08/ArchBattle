package http

import (
	"encoding/json"
	stdhttp "net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type RouterParams struct {
	Middleware          *Middleware
	AuthHandler         *AuthHandler
	MatchHandler        *MatchHandler
	DailyHandler        *DailyHandler
	DiscussionHandler   *DiscussionHandler
	QuestionHandler     *QuestionHandler
	LeaderboardHandler  *LeaderboardHandler
	AdminHandler        *AdminHandler
	UserHandler         *UserHandler
	TutorHandler        *TutorHandler
	WSGateway           stdhttp.Handler
	MetricsHandler      stdhttp.Handler
	AllowedOrigins      []string
}

func NewRouter(params RouterParams) stdhttp.Handler {
	router := chi.NewRouter()

	corsHandler := cors.Handler(cors.Options{
		AllowedOrigins:   params.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(corsHandler)
	if params.Middleware != nil {
		router.Use(params.Middleware.RequestLogger)
	}

	router.Get("/health", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	if params.MetricsHandler != nil {
		router.Handle("/metrics", params.MetricsHandler)
	}

	// /ws is exempt from the 30s timeout because WebSocket connections are long-lived.
	if params.WSGateway != nil {
		router.Handle("/ws", params.WSGateway)
	}

	// Admin AI routes with 2min timeout (must be registered before 30s group to take precedence).
	if params.Middleware != nil {
		router.Group(func(r chi.Router) {
			r.Use(params.Middleware.Authenticated)
			r.Use(params.Middleware.AdminOnly)
			r.Use(chimiddleware.Timeout(2 * time.Minute))
			r.Post("/admin/ai/draft-question", params.AdminHandler.DraftQuestion)
			r.Post("/admin/daily-challenge/summary", params.AdminHandler.GenerateDailySummary)
		})
	}

	// All HTTP API routes share the 30s request timeout.
	router.Group(func(r chi.Router) {
		r.Use(chimiddleware.Timeout(30 * time.Second))

		r.Route("/auth", func(auth chi.Router) {
			auth.Post("/register", params.AuthHandler.Register)
			auth.Post("/login", params.AuthHandler.Login)
			if params.Middleware != nil {
				auth.With(params.Middleware.Authenticated).Post("/logout", params.AuthHandler.Logout)
			}
		})

		r.Get("/leaderboard", params.LeaderboardHandler.Get)

		if params.Middleware != nil {
			r.Group(func(api chi.Router) {
				api.Use(params.Middleware.Authenticated)
				api.Post("/match/queue", params.MatchHandler.Queue)
				api.Delete("/match/queue", params.MatchHandler.Dequeue)
				api.Get("/daily-challenge", params.DailyHandler.GetChallenge)
				api.Post("/daily-submit", params.DailyHandler.Submit)
				api.Get("/daily-share-card", params.DailyHandler.ShareCard)
				api.Route("/daily-challenge/{date}/discussion", func(d chi.Router) {
					d.Get("/", params.DiscussionHandler.List)
					d.Post("/", params.DiscussionHandler.Create)
					d.Post("/{id}/upvote", params.DiscussionHandler.Upvote)
				})
				api.Post("/questions/{id}/dispute", params.QuestionHandler.Dispute)
				api.Get("/users/me", params.UserHandler.Me)
				if params.TutorHandler != nil {
					api.Post("/api/tutor", params.TutorHandler.Handle)
				}
				api.Route("/admin", func(admin chi.Router) {
					admin.Use(params.Middleware.AdminOnly)
					admin.Get("/questions", params.AdminHandler.ListDrafts)
					admin.Post("/questions", params.AdminHandler.CreateQuestion)
					admin.Patch("/questions/{id}", params.AdminHandler.UpdateQuestionStatus)
					admin.Get("/disputes", params.AdminHandler.ListDisputes)
					admin.Post("/daily-challenge/publish", params.AdminHandler.PublishDaily)
					// AI draft and summary registered above with 2min timeout.
				})
			})
		}
	})

	return router
}
