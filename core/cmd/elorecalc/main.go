// elorecalc is a maintenance cron job that syncs Postgres ELO values into the
// Redis leaderboard sorted sets. Run it periodically (e.g. daily) to repair
// any drift caused by crashes, missed updates, or manual DB corrections.
//
// Usage:
//
//	DATABASE_URL=... REDIS_URL=... go run ./core/cmd/elorecalc
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	databaseURL := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/archbattle?sslmode=disable")
	redisURL := env("REDIS_URL", "redis://localhost:6379/0")

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		logger.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisOptions, err := goredis.ParseURL(redisURL)
	if err != nil {
		logger.Error("parse redis url", "error", err)
		os.Exit(1)
	}
	redisClient := goredis.NewClient(redisOptions)
	defer redisClient.Close()

	updated, err := syncELO(ctx, logger, pool, redisClient)
	if err != nil {
		logger.Error("elo recalc failed", "error", err)
		os.Exit(1)
	}
	logger.Info("elo recalc complete", "users_synced", updated)
}

type userELORow struct {
	ID        uuid.UUID
	Tier      string
	JuniorELO int
	SeniorELO int
	StaffELO  int
}

// syncELO reads every user's current ELO from Postgres and writes it into the
// Redis global leaderboard sorted sets, overwriting any stale scores.
func syncELO(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool, rc *goredis.Client) (int, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, tier, junior_elo, senior_elo, staff_elo
		FROM users
		ORDER BY id
	`)
	if err != nil {
		return 0, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []userELORow
	for rows.Next() {
		var u userELORow
		if err := rows.Scan(&u.ID, &u.Tier, &u.JuniorELO, &u.SeniorELO, &u.StaffELO); err != nil {
			return 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate users: %w", err)
	}

	week := isoWeekKey(time.Now().UTC())
	pipe := rc.Pipeline()
	for _, u := range users {
		// Write each tier's absolute ELO into both the global and current-week leaderboard.
		for _, entry := range []struct {
			tier string
			elo  int
		}{
			{"junior", u.JuniorELO},
			{"senior", u.SeniorELO},
			{"staff", u.StaffELO},
		} {
			globalKey := fmt.Sprintf("lb:global:%s", entry.tier)
			weeklyKey := fmt.Sprintf("lb:weekly:%s:%s", entry.tier, week)
			pipe.ZAdd(ctx, globalKey, goredis.Z{Score: float64(entry.elo), Member: u.ID.String()})
			pipe.ZAdd(ctx, weeklyKey, goredis.Z{Score: float64(entry.elo), Member: u.ID.String()})
			pipe.Expire(ctx, weeklyKey, 7*24*time.Hour)
		}
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("pipeline exec: %w", err)
	}
	logger.Info("synced users", "count", len(users), "week", week)
	return len(users), nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func isoWeekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}
