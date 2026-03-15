package main

import (
	"context"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	inboundhttp "github.com/radhakrishna/archbattle/ai-service/internal/adapter/inbound/http"
	"github.com/radhakrishna/archbattle/ai-service/internal/adapter/outbound/openai"
	postgresadapter "github.com/radhakrishna/archbattle/ai-service/internal/adapter/outbound/postgres"
	redisadapter "github.com/radhakrishna/archbattle/ai-service/internal/adapter/outbound/redis"
	"github.com/radhakrishna/archbattle/ai-service/internal/config"
	"github.com/radhakrishna/archbattle/ai-service/internal/domain/calibrator"
	"github.com/radhakrishna/archbattle/ai-service/internal/domain/drafter"
	"github.com/radhakrishna/archbattle/ai-service/internal/domain/summary"
	"github.com/radhakrishna/archbattle/ai-service/internal/domain/tutor"
	"github.com/radhakrishna/archbattle/ai-service/internal/domain/variation"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	redisClient, err := redisadapter.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("connect redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	pgPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	llm := openai.NewClient(cfg.OpenAIAPIKey)
	limiter := redisadapter.NewRateLimiter(redisClient)
	sessionLogger := postgresadapter.NewSessionLogger(pgPool)

	tutorService := tutor.NewService(llm, limiter, sessionLogger, cfg.FreeTutorLimitPerDay)
	drafterService := drafter.NewService(llm)
	variationService := variation.NewService(llm)
	calibratorService := calibrator.NewService(llm)
	summaryService := summary.NewService(llm)

	router := inboundhttp.NewRouter(inboundhttp.RouterParams{
		TutorHandler:     inboundhttp.NewTutorHandler(tutorService),
		DraftHandler:     inboundhttp.NewDraftHandler(drafterService),
		VariationHandler: inboundhttp.NewVariationHandler(variationService),
		CalibrateHandler: inboundhttp.NewCalibrateHandler(calibratorService),
		SummaryHandler:   inboundhttp.NewSummaryHandler(summaryService),
		AllowedOrigins:   cfg.AllowedOrigins,
		InternalSecret:   cfg.InternalSecret,
	})

	server := &stdhttp.Server{Addr: ":" + cfg.HTTPPort, Handler: router, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		logger.Info("ai service listening", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != stdhttp.ErrServerClosed {
			logger.Error("run ai http server", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
}
