package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

type questionSeed struct {
	ID             uuid.UUID
	Mode           string
	Topic          string
	DifficultyTier string
	Prompt         string
	Options        []string
	CorrectAnswers []int
	Rationale      string
}

func main() {
	questionCount := flag.Int("questions", 20, "number of questions to seed")
	dailyCount := flag.Int("daily", 7, "number of daily challenges to seed")
	flag.Parse()

	ctx := context.Background()
	databaseURL := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/archbattle?sslmode=disable")
	redisURL := env("REDIS_URL", "redis://localhost:6379/0")

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer pool.Close()

	redisOptions, err := goredis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("parse redis url: %v", err)
	}
	redisClient := goredis.NewClient(redisOptions)
	defer redisClient.Close()

	users := seedUsers(ctx, pool)
	questions := seedQuestions(ctx, pool, *questionCount)
	seedDailyChallenges(ctx, pool, questions, *dailyCount)
	seedLeaderboards(ctx, redisClient, users)

	log.Printf("seed complete: %d users, %d questions, %d daily challenge sets", len(users), len(questions), *dailyCount)
}

func seedUsers(ctx context.Context, pool *pgxpool.Pool) []uuid.UUID {
	users := []struct {
		Username string
		Email    string
		Tier     string
		ELO      int
	}{
		{"junior_amy", "amy@archbattle.dev", "junior", 1000},
		{"junior_ben", "ben@archbattle.dev", "junior", 1040},
		{"senior_chris", "chris@archbattle.dev", "senior", 1520},
		{"senior_dana", "dana@archbattle.dev", "senior", 1490},
		{"staff_evan", "evan@archbattle.dev", "staff", 1710},
	}
	ids := make([]uuid.UUID, 0, len(users))
	for _, user := range users {
		id := uuid.New()
		_, err := pool.Exec(ctx, `
            INSERT INTO users (id, username, email, password_hash, role, tier, junior_elo, senior_elo, staff_elo, matches_played, current_streak, longest_streak, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 0, 0, 0, NOW())
            ON CONFLICT (email) DO NOTHING
        `, id, user.Username, user.Email, "$2a$10$P5sYjM1Qz9S8d3h2V8jJYupxH5L6P6LFfBvL8Y9X7V0gQxW0fQbK2", "user", user.Tier, user.ELO, user.ELO, user.ELO)
		if err != nil {
			log.Fatalf("insert user %s: %v", user.Email, err)
		}
		ids = append(ids, id)
	}
	return ids
}

// curated is a hand-crafted set of realistic software architecture questions.
// Additional questions are generated generically once the curated list is exhausted.
var curated = []questionSeed{
	{
		Mode: "fff", Topic: "caching", DifficultyTier: "junior",
		Prompt:         "Your read-heavy API returns the same catalog data to all users. Which caching strategy best reduces database load?",
		Options:        []string{"Write-through cache with Redis", "Client-side cookie caching", "No caching — always read from DB", "Write-behind cache"},
		CorrectAnswers: []int{0},
		Rationale:      "Write-through cache keeps Redis and the DB in sync on every write, so reads can be served from Redis with low latency and reduced DB pressure.",
	},
	{
		Mode: "fff", Topic: "caching", DifficultyTier: "senior",
		Prompt:         "Cache invalidation is hard. You have a write-through cache and a product update triggers a burst of cache misses. What is the best mitigation?",
		Options:        []string{"Increase cache TTL to one week", "Use a cache-aside pattern with lazy population", "Invalidate all cache keys on any write", "Disable caching during high write periods"},
		CorrectAnswers: []int{1},
		Rationale:      "Cache-aside (lazy population) only fetches data on a miss, reducing thundering herd. Combined with probabilistic early expiration it limits stampede effects.",
	},
	{
		Mode: "fff", Topic: "queues", DifficultyTier: "junior",
		Prompt:         "You need to decouple a payment processor from an order service. Which pattern best fits this use case?",
		Options:        []string{"Direct HTTP call with retry logic", "Message queue (e.g. SQS or RabbitMQ)", "Shared database table polled by both services", "Synchronous gRPC stream"},
		CorrectAnswers: []int{1},
		Rationale:      "Message queues decouple producers from consumers, support retries, backpressure, and allow the payment processor to scale independently.",
	},
	{
		Mode: "fff", Topic: "queues", DifficultyTier: "senior",
		Prompt:         "A consumer crashes mid-processing of a queue message. What mechanism prevents message loss while avoiding duplicate processing?",
		Options:        []string{"At-least-once delivery with idempotency keys on the consumer", "At-most-once delivery so the message is dropped on crash", "Exactly-once delivery using distributed transactions across producer and consumer", "No acknowledgement — fire and forget"},
		CorrectAnswers: []int{0},
		Rationale:      "At-least-once delivery retries on consumer failure. Idempotency keys on the consumer side make re-processing safe, effectively achieving at-least-once semantics with safe duplicate handling.",
	},
	{
		Mode: "fff", Topic: "storage", DifficultyTier: "junior",
		Prompt:         "You need to store user-uploaded profile photos accessed frequently by millions of users. Which storage approach is most appropriate?",
		Options:        []string{"Store as BLOBs in PostgreSQL", "Object storage (e.g. S3) with a CDN in front", "Local disk on the API server", "Store as base64 in a Redis hash"},
		CorrectAnswers: []int{1},
		Rationale:      "Object storage is designed for binary files and scales cheaply. A CDN caches images at edge nodes close to users, dramatically reducing latency and origin load.",
	},
	{
		Mode: "fff", Topic: "storage", DifficultyTier: "senior",
		Prompt:         "When would you choose event sourcing over a traditional CRUD data model?",
		Options:        []string{"When you need a complete, immutable audit trail and the ability to replay state", "When you want to minimize storage costs", "When queries require complex ad-hoc joins across many entities", "When write throughput is the primary constraint"},
		CorrectAnswers: []int{0},
		Rationale:      "Event sourcing stores every state change as an immutable event. This provides a full audit log and allows replaying history to rebuild any past state, making it ideal for financial ledgers and compliance-sensitive domains.",
	},
	{
		Mode: "fff", Topic: "rate-limiting", DifficultyTier: "junior",
		Prompt:         "Which algorithm distributes request capacity most smoothly over time, avoiding bursty behavior?",
		Options:        []string{"Fixed window counter", "Token bucket", "Leaky bucket", "Random rejection"},
		CorrectAnswers: []int{2},
		Rationale:      "The leaky bucket enforces a constant outflow rate, smoothing bursts. Token bucket allows controlled bursts up to the bucket size, but leaky bucket produces the most uniform flow.",
	},
	{
		Mode: "fff", Topic: "rate-limiting", DifficultyTier: "senior",
		Prompt:         "You implement a sliding window rate limiter using Redis. A Redis INCR call succeeds but the subsequent EXPIRE call fails. What is the risk?",
		Options:        []string{"The counter is never decremented, permanently blocking the user", "The key has no TTL — the counter accumulates forever, blocking the user indefinitely", "Redis returns a stale value on the next request", "No risk — the key will expire at the default Redis TTL"},
		CorrectAnswers: []int{1},
		Rationale:      "Without a TTL the counter key lives forever. The rate limiter will permanently block the user once the count exceeds the limit. The fix is to use a Redis Lua script or pipeline to atomically set both the value and TTL.",
	},
	{
		Mode: "fff", Topic: "observability", DifficultyTier: "junior",
		Prompt:         "Your service latency spiked for 5 minutes but all health checks passed. Which observability signal helps you understand WHY it happened?",
		Options:        []string{"Uptime percentage SLA", "P99 latency histogram with distributed traces", "CPU average over the last 24 hours", "Number of deployments this week"},
		CorrectAnswers: []int{1},
		Rationale:      "A P99 latency histogram reveals tail latency spikes invisible in averages. Distributed traces help you pinpoint which service call or DB query caused the spike.",
	},
	{
		Mode: "fff", Topic: "observability", DifficultyTier: "staff",
		Prompt:         "You observe a gradual increase in error rate correlated with memory growth over several days. What is the most likely root cause and diagnostic approach?",
		Options:        []string{"A network partition — add more replicas", "A memory leak — use heap profiling and correlate with GC pause metrics", "Too many indexes in PostgreSQL", "Insufficient CPU for the workload"},
		CorrectAnswers: []int{1},
		Rationale:      "Gradual memory growth correlated with errors suggests a leak. Heap profiling (pprof in Go, jmap in JVM) identifies the allocation site. Correlating with GC pause metrics confirms the GC is unable to reclaim memory fast enough.",
	},
}

func seedQuestions(ctx context.Context, pool *pgxpool.Pool, total int) []questionSeed {
	topics := []string{"caching", "queues", "storage", "rate-limiting", "observability"}
	tiers := []string{"junior", "senior", "staff"}
	questions := make([]questionSeed, 0, total)
	for idx := 0; idx < total; idx++ {
		var question questionSeed
		if idx < len(curated) {
			question = curated[idx]
			question.ID = uuid.New()
		} else {
			question = questionSeed{
				ID:             uuid.New(),
				Mode:           "fff",
				Topic:          topics[idx%len(topics)],
				DifficultyTier: tiers[idx%len(tiers)],
				Prompt:         fmt.Sprintf("In a %s system under sudden load, what is the best trade-off strategy for %s?", tiers[idx%len(tiers)], topics[idx%len(topics)]),
				Options:        []string{"Add redundancy layers", "Apply backpressure upstream", "Degrade gracefully with circuit breakers", "Scale horizontally and rebalance"},
				CorrectAnswers: []int{idx % 4},
				Rationale:      "Combining backpressure, graceful degradation, and circuit breakers achieves resilience without cascading failures.",
			}
		}
		optionJSON, _ := json.Marshal(question.Options)
		correctJSON, _ := json.Marshal(question.CorrectAnswers)
		_, err := pool.Exec(ctx, `
            INSERT INTO questions (id, mode, topic, difficulty_tier, prompt, options, correct_answers, rationale, dispute_count, pilot_attempts, pilot_dispute_rate, is_active, daily_eligible, status, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, 0, 0, TRUE, TRUE, 'live', NOW())
            ON CONFLICT (id) DO NOTHING
        `, question.ID, question.Mode, question.Topic, question.DifficultyTier, question.Prompt, optionJSON, correctJSON, question.Rationale)
		if err != nil {
			log.Fatalf("insert question %d: %v", idx+1, err)
		}
		questions = append(questions, question)
	}
	return questions
}

func seedDailyChallenges(ctx context.Context, pool *pgxpool.Pool, questions []questionSeed, count int) {
	for idx := 0; idx < count; idx++ {
		questionIDs := []uuid.UUID{}
		for offset := 0; offset < 3; offset++ {
			questionIDs = append(questionIDs, questions[(idx+offset)%len(questions)].ID)
		}
		_, err := pool.Exec(ctx, `
            INSERT INTO daily_challenges (id, challenge_date, question_ids, theme, ai_summary, summary_reviewed, published_at, created_at)
            VALUES ($1, $2, $3, $4, $5, TRUE, NOW(), NOW())
            ON CONFLICT (challenge_date) DO NOTHING
        `, uuid.New(), time.Now().UTC().AddDate(0, 0, idx), questionIDs, fmt.Sprintf("Theme %d", idx+1), "A focused daily challenge around trade-off analysis.")
		if err != nil {
			log.Fatalf("insert daily challenge %d: %v", idx+1, err)
		}
	}
}

func seedLeaderboards(ctx context.Context, redisClient *goredis.Client, userIDs []uuid.UUID) {
	scores := []float64{1000, 1040, 1520, 1490, 1710}
	week := isoWeekKey(time.Now().UTC())
	tiers := []string{"junior", "junior", "senior", "senior", "staff"}
	for idx, userID := range userIDs {
		_ = redisClient.ZAdd(ctx, fmt.Sprintf("lb:global:%s", tiers[idx]), goredis.Z{Score: scores[idx], Member: userID.String()}).Err()
		_ = redisClient.ZAdd(ctx, fmt.Sprintf("lb:weekly:%s:%s", tiers[idx], week), goredis.Z{Score: scores[idx], Member: userID.String()}).Err()
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func isoWeekKey(now time.Time) string {
	year, week := now.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}
