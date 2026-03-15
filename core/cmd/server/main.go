package main

import (
	"context"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	inboundcron "github.com/radhakrishna/archbattle/core/internal/adapter/inbound/cron"
	inboundhttp "github.com/radhakrishna/archbattle/core/internal/adapter/inbound/http"
	inboundws "github.com/radhakrishna/archbattle/core/internal/adapter/inbound/ws"
	outboundai "github.com/radhakrishna/archbattle/core/internal/adapter/outbound/ai"
	outboundapp "github.com/radhakrishna/archbattle/core/internal/adapter/outbound/app"
	outboundpostgres "github.com/radhakrishna/archbattle/core/internal/adapter/outbound/postgres"
	outboundredis "github.com/radhakrishna/archbattle/core/internal/adapter/outbound/redis"
	"github.com/radhakrishna/archbattle/core/internal/config"
	domaindaily "github.com/radhakrishna/archbattle/core/internal/domain/daily"
	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
	domainquestion "github.com/radhakrishna/archbattle/core/internal/domain/question"
	domainroom "github.com/radhakrishna/archbattle/core/internal/domain/room"
	"github.com/radhakrishna/archbattle/core/internal/observability"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pgPool, err := outboundpostgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pgPool.Close()

	migrationsDir := filepath.Join("migrations")
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		if err := outboundpostgres.RunMigrations(cfg.DatabaseURL, migrationsDir); err != nil {
			logger.Error("run migrations", "error", err)
			os.Exit(1)
		}
		logger.Info("migrations completed")
		return
	}
	if err := outboundpostgres.RunMigrations(cfg.DatabaseURL, migrationsDir); err != nil {
		logger.Error("run startup migrations", "error", err)
		os.Exit(1)
	}

	redisClient, err := outboundredis.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("connect redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	metrics := observability.NewMetrics("archbattle", prometheus.DefaultRegisterer)

	userRepo := outboundpostgres.NewUserRepo(pgPool)
	matchRepo := outboundpostgres.NewMatchRepo(pgPool)
	submissionRepo := outboundpostgres.NewSubmissionRepo(pgPool)
	questionRepo := outboundpostgres.NewQuestionRepo(pgPool)
	dailyRepo := outboundpostgres.NewDailyRepo(pgPool)

	matchStateStore := outboundredis.NewMatchStateStore(redisClient)
	answerStore := outboundredis.NewAnswerStore(redisClient)
	eventPublisher := outboundredis.NewEventPublisher(redisClient, cfg.MatchStreamTTL)
	roomStore := outboundredis.NewRoomStore(redisClient)
	dailyCacheStore := outboundredis.NewDailyCacheStore(redisClient)
	dailyLeaderboardStore := outboundredis.NewDailyLeaderboardStore(redisClient)

	aiClient := outboundai.NewClient(cfg.AIServiceURL, cfg.AIInternalSecret)

	questionService := domainquestion.NewService(questionRepo, cfg.DisputeThreshold, aiClient)
	dailyService := domaindaily.NewService(dailyRepo, dailyCacheStore, dailyLeaderboardStore, cfg.StreakGraceHours)
	metricsTransitionRecorder := &metricsTransitionRecorder{metrics: metrics}
	matchService := domainmatch.NewService(matchRepo, submissionRepo, matchStateStore, answerStore, eventPublisher, nil, questionService, cfg.MatchStreamTTL, metricsTransitionRecorder)

	wsGateway := inboundws.NewGateway(matchService, nil, questionService, eventPublisher, nil, cfg.AllowedOrigins, logger)
	wsGateway.SetMetrics(metrics)
	streamReader := outboundredis.NewMatchStreamReader(eventPublisher, wsGateway, cfg.MatchBlockTimeout)
	wsGateway.SetStreamReader(streamReader)
	matchService.SetBroadcaster(wsGateway)

	roomStatusReader := outboundapp.NewRoomStatusReader(matchRepo, matchStateStore)
	roomService := domainroom.NewService(roomStore, matchService, matchService, roomStatusReader, wsGateway)

	router := inboundhttp.NewRouter(inboundhttp.RouterParams{
		DailyHandler:   inboundhttp.NewDailyHandler(dailyService),
		PlayerHandler:  inboundhttp.NewPlayerHandler(userRepo),
		RoomHandler:    inboundhttp.NewRoomHandler(roomService),
		TutorHandler:   inboundhttp.NewTutorHandler(aiClient),
		WSGateway:      wsGateway,
		MetricsHandler: promhttp.Handler(),
		AllowedOrigins: cfg.AllowedOrigins,
	})

	if bufferDays, err := dailyService.BufferDays(ctx, time.Now().UTC()); err == nil {
		metrics.DailyChallengeBufferDays.Set(float64(bufferDays))
	}

	server := &stdhttp.Server{Addr: ":" + cfg.HTTPPort, Handler: router, ReadHeaderTimeout: 10 * time.Second}
	scheduler := inboundcron.NewScheduler(dailyService, dailyRepo, logger)

	go func() {
		logger.Info("core service listening", "port", cfg.HTTPPort)
		if err := server.ListenAndServe(); err != nil && err != stdhttp.ErrServerClosed {
			logger.Error("run http server", "error", err)
			cancel()
		}
	}()

	go func() {
		if err := scheduler.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("run scheduler", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
}

type metricsTransitionRecorder struct {
	metrics *observability.Metrics
}

func (m *metricsTransitionRecorder) RecordMatchStateTransitionError() {
	m.metrics.MatchStateTransitionError.Inc()
}
