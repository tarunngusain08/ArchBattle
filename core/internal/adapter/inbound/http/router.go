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
	DailyHandler   *DailyHandler
	PlayerHandler  *PlayerHandler
	RoomHandler    *RoomHandler
	TutorHandler   *TutorHandler
	WSGateway      stdhttp.Handler
	MetricsHandler stdhttp.Handler
	AllowedOrigins []string
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

	router.Get("/health", func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	if params.MetricsHandler != nil {
		router.Handle("/metrics", params.MetricsHandler)
	}

	if params.WSGateway != nil {
		router.Handle("/ws", params.WSGateway)
	}

	router.Group(func(r chi.Router) {
		r.Use(chimiddleware.Timeout(30 * time.Second))

		r.Post("/join", params.PlayerHandler.Join)
		r.Post("/rooms", params.RoomHandler.CreateRoom)
		r.Post("/rooms/{code}/join", params.RoomHandler.JoinRoom)
		r.Get("/rooms/{code}", params.RoomHandler.GetRoomStatus)

		r.Get("/daily-challenge", params.DailyHandler.GetChallenge)
		r.Post("/daily-submit", params.DailyHandler.Submit)
		r.Get("/daily-share-card", params.DailyHandler.ShareCard)

		if params.TutorHandler != nil {
			r.Post("/api/tutor", params.TutorHandler.Handle)
		}
	})

	return router
}
