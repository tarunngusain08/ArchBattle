package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv               string
	HTTPPort             string
	WSPath               string
	DatabaseURL          string
	RedisURL             string
	JWTSecret            string
	AIServiceURL         string
	AIInternalSecret     string
	AllowedOrigins       []string
	DisputeThreshold     float64
	StreakGraceHours     int
	FreeTutorLimitPerDay int
	FreeMatchLimitPerDay int
	MatchStreamTTL       time.Duration
	MatchBlockTimeout    time.Duration
}

func Load() Config {
	return Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		HTTPPort:             getEnv("CORE_HTTP_PORT", "8080"),
		WSPath:               getEnv("CORE_WS_PATH", "/ws"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/archbattle?sslmode=disable"),
		RedisURL:             getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:            getEnv("JWT_SECRET", "change-me"),
		AIServiceURL:         getEnv("AI_SERVICE_URL", "http://localhost:8081"),
		AIInternalSecret:     os.Getenv("AI_INTERNAL_SECRET"),
		AllowedOrigins:       splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:5173")),
		DisputeThreshold:     getEnvFloat("DISPUTE_THRESHOLD", 0.08),
		StreakGraceHours:     getEnvInt("STREAK_GRACE_HOURS", 48),
		FreeTutorLimitPerDay: getEnvInt("FREE_TUTOR_LIMIT_PER_DAY", 3),
		FreeMatchLimitPerDay: getEnvInt("FREE_MATCH_LIMIT_PER_DAY", 5),
		MatchStreamTTL:       10 * time.Minute,
		MatchBlockTimeout:    2 * time.Second,
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	raw := strings.Split(value, ",")
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
