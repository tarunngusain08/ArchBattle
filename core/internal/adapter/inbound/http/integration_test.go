//go:build integration

package http

import (
	"os"
	"testing"
)

// Integration tests require DATABASE_URL and REDIS_URL to be set.
// Run with: go test -tags=integration ./internal/adapter/inbound/http/...
//
// Example: docker-compose up -d redis postgres && \
//   go test -tags=integration ./core/internal/adapter/inbound/http/... -v
func TestIntegration_RequiresEnv(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	redisURL := os.Getenv("REDIS_URL")
	if dbURL == "" || redisURL == "" {
		t.Skip("DATABASE_URL and REDIS_URL required for integration tests. Start infra with: docker-compose up -d redis postgres")
	}
	// Full integration tests would wire up the app (postgres, redis, services)
	// and exercise: auth register/login, match queue, daily submit, reconnect.
	// Placeholder for future implementation.
	t.Log("integration test suite ready - add full flow tests when infra is available")
}
