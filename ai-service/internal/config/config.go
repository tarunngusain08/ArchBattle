package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv               string
	HTTPPort             string
	RedisURL             string
	DatabaseURL          string
	GeminiAPIKey         string
	InternalSecret       string
	AllowedOrigins       []string
	FreeTutorLimitPerDay int
}

func Load() Config {
	return Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		HTTPPort:             getEnv("AI_HTTP_PORT", "8081"),
		RedisURL:             getEnv("REDIS_URL", "redis://localhost:6379/0"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/archbattle?sslmode=disable"),
		GeminiAPIKey:         os.Getenv("GEMINI_API_KEY"),
		InternalSecret:       os.Getenv("AI_INTERNAL_SECRET"),
		AllowedOrigins:       splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:5173")),
		FreeTutorLimitPerDay: getEnvInt("FREE_TUTOR_LIMIT_PER_DAY", 3),
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

func splitCSV(value string) []string {
	raw := strings.Split(value, ",")
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
