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
	questionCount := flag.Int("questions", 100, "number of questions to seed")
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

// curated is a hand-crafted set of ~100 scenario-based architecture questions.
// From example-questions.md and requirements.md. AI generation is primary; these are fallback.
var curated = []questionSeed{
	// --- From example-questions.md (1-10) ---
	{
		Mode: "fff", Topic: "caching", DifficultyTier: "junior",
		Prompt:         "Your ecommerce platform experiences a flash sale where traffic increases from 3K requests/sec to 120K requests/sec within 30 seconds. Product catalog reads dominate traffic and the primary Postgres database CPU reaches 95%.\n\nWhat architecture change should be prioritized first to reduce database load?",
		Options:        []string{"Add read replicas to the Postgres database", "Introduce Redis caching for product catalog queries", "Increase Postgres instance size", "Add additional API servers"},
		CorrectAnswers: []int{1},
		Rationale:      "The workload is read-heavy, so introducing Redis caching drastically reduces DB queries. Scaling the database or API servers does not reduce the read amplification problem.",
	},
	{
		Mode: "fff", Topic: "caching", DifficultyTier: "senior",
		Prompt:         "A social media platform stores trending posts in Redis. One post goes viral and becomes a hot key, receiving 200K reads/sec, causing Redis CPU saturation.\n\nWhat is the best strategy to mitigate this hot key issue?",
		Options:        []string{"Use client-side caching for that key", "Replicate Redis to additional nodes and shard reads", "Increase Redis memory allocation", "Reduce TTL of the hot key"},
		CorrectAnswers: []int{1},
		Rationale:      "Hot keys cause uneven load. Replicating Redis nodes and distributing reads prevents one node from becoming overloaded.",
	},
	{
		Mode: "fff", Topic: "queues", DifficultyTier: "junior",
		Prompt:         "An order processing pipeline uses Kafka with 4 partitions. During peak traffic, consumer lag grows rapidly and orders are processed minutes late.\n\nWhat should be done first?",
		Options:        []string{"Increase number of Kafka partitions", "Increase retention time", "Restart consumers", "Increase message size limit"},
		CorrectAnswers: []int{0},
		Rationale:      "Consumer throughput is limited by partition parallelism. Increasing partitions allows more consumers to process events concurrently.",
	},
	{
		Mode: "fff", Topic: "storage", DifficultyTier: "junior",
		Prompt:         "Users in Asia report high latency while streaming videos hosted on servers in the US.\n\nWhat architecture change will most improve performance?",
		Options:        []string{"Deploy CDN edge caches globally", "Increase bandwidth on US servers", "Compress video files further", "Increase server CPU capacity"},
		CorrectAnswers: []int{0},
		Rationale:      "CDNs cache content closer to users, reducing latency and international bandwidth requirements.",
	},
	{
		Mode: "fff", Topic: "queues", DifficultyTier: "senior",
		Prompt:         "A payment service depends on a downstream fraud-check API. The API becomes slow, causing your service to retry requests aggressively, increasing load further.\n\nWhich mechanism should be implemented?",
		Options:        []string{"Circuit breaker pattern", "Increase retry count", "Scale API servers horizontally", "Cache fraud results permanently"},
		CorrectAnswers: []int{0},
		Rationale:      "Circuit breakers prevent retry storms by temporarily halting calls when failure thresholds are exceeded.",
	},
	{
		Mode: "fff", Topic: "storage", DifficultyTier: "senior",
		Prompt:         "A messaging platform stores billions of messages in a distributed database. Messages are currently sharded by user_id, but some users send millions of messages per day.\n\nWhat is the best alternative sharding strategy?",
		Options:        []string{"Shard by message timestamp", "Shard by conversation_id", "Shard by message length", "Use a single database cluster"},
		CorrectAnswers: []int{1},
		Rationale:      "Conversation-level sharding distributes load evenly and preserves query locality for conversations.",
	},
	{
		Mode: "fff", Topic: "observability", DifficultyTier: "junior",
		Prompt:         "A chat application uses WebSocket servers. As concurrent users grow to 500K, some servers exceed connection limits.\n\nWhat architecture change is most effective?",
		Options:        []string{"Introduce a load balancer with sticky sessions", "Replace WebSockets with HTTP polling", "Increase database capacity", "Cache chat messages in Redis"},
		CorrectAnswers: []int{0},
		Rationale:      "Sticky sessions allow consistent routing of WebSocket connections and enable horizontal scaling across servers.",
	},
	{
		Mode: "fff", Topic: "caching", DifficultyTier: "senior",
		Prompt:         "A popular API endpoint caches results in Redis with a TTL of 5 minutes. When the key expires, thousands of requests simultaneously hit the database.\n\nWhich solution prevents this issue?",
		Options:        []string{"Randomized TTL with jitter", "Reduce TTL to 1 minute", "Increase DB connection pool", "Use synchronous DB queries"},
		CorrectAnswers: []int{0},
		Rationale:      "Adding TTL jitter prevents synchronized cache expiration.",
	},
	{
		Mode: "fff", Topic: "storage", DifficultyTier: "junior",
		Prompt:         "A gaming platform maintains a global leaderboard for 10M users. Queries request top 100 players frequently.\n\nWhich storage approach is best?",
		Options:        []string{"Redis Sorted Sets", "Postgres table with index", "Flat file storage", "Distributed object storage"},
		CorrectAnswers: []int{0},
		Rationale:      "Redis Sorted Sets provide O(log N) ranking and efficient top-K queries.",
	},
	{
		Mode: "fff", Topic: "rate-limiting", DifficultyTier: "junior",
		Prompt:         "Your public API experiences abusive traffic spikes from a small number of clients.\n\nWhat is the most scalable way to enforce rate limits?",
		Options:        []string{"Per-client token bucket using Redis", "Increase server CPU", "Block all traffic temporarily", "Add additional API servers"},
		CorrectAnswers: []int{0},
		Rationale:      "Token bucket algorithms using Redis counters allow distributed rate limiting across multiple servers.",
	},
	// --- Original curated (11-20) ---
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
		Rationale:      "At-least-once delivery retries on consumer failure. Idempotency keys on the consumer side make re-processing safe.",
	},
	{
		Mode: "fff", Topic: "storage", DifficultyTier: "junior",
		Prompt:         "You need to store user-uploaded profile photos accessed frequently by millions of users. Which storage approach is most appropriate?",
		Options:        []string{"Store as BLOBs in PostgreSQL", "Object storage (e.g. S3) with a CDN in front", "Local disk on the API server", "Store as base64 in a Redis hash"},
		CorrectAnswers: []int{1},
		Rationale:      "Object storage is designed for binary files and scales cheaply. A CDN caches images at edge nodes close to users.",
	},
	{
		Mode: "fff", Topic: "storage", DifficultyTier: "senior",
		Prompt:         "When would you choose event sourcing over a traditional CRUD data model?",
		Options:        []string{"When you need a complete, immutable audit trail and the ability to replay state", "When you want to minimize storage costs", "When queries require complex ad-hoc joins across many entities", "When write throughput is the primary constraint"},
		CorrectAnswers: []int{0},
		Rationale:      "Event sourcing stores every state change as an immutable event. This provides a full audit log and allows replaying history to rebuild any past state.",
	},
	{
		Mode: "fff", Topic: "rate-limiting", DifficultyTier: "junior",
		Prompt:         "Which algorithm distributes request capacity most smoothly over time, avoiding bursty behavior?",
		Options:        []string{"Fixed window counter", "Token bucket", "Leaky bucket", "Random rejection"},
		CorrectAnswers: []int{2},
		Rationale:      "The leaky bucket enforces a constant outflow rate, smoothing bursts. Token bucket allows controlled bursts but leaky bucket produces the most uniform flow.",
	},
	{
		Mode: "fff", Topic: "rate-limiting", DifficultyTier: "senior",
		Prompt:         "You implement a sliding window rate limiter using Redis. A Redis INCR call succeeds but the subsequent EXPIRE call fails. What is the risk?",
		Options:        []string{"The counter is never decremented, permanently blocking the user", "The key has no TTL — the counter accumulates forever, blocking the user indefinitely", "Redis returns a stale value on the next request", "No risk — the key will expire at the default Redis TTL"},
		CorrectAnswers: []int{1},
		Rationale:      "Without a TTL the counter key lives forever. The rate limiter will permanently block the user once the count exceeds the limit.",
	},
	{
		Mode: "fff", Topic: "observability", DifficultyTier: "junior",
		Prompt:         "Your service latency spiked for 5 minutes but all health checks passed. Which observability signal helps you understand WHY it happened?",
		Options:        []string{"Uptime percentage SLA", "P99 latency histogram with distributed traces", "CPU average over the last 24 hours", "Number of deployments this week"},
		CorrectAnswers: []int{1},
		Rationale:      "A P99 latency histogram reveals tail latency spikes invisible in averages. Distributed traces help pinpoint which service call or DB query caused the spike.",
	},
	{
		Mode: "fff", Topic: "observability", DifficultyTier: "staff",
		Prompt:         "You observe a gradual increase in error rate correlated with memory growth over several days. What is the most likely root cause and diagnostic approach?",
		Options:        []string{"A network partition — add more replicas", "A memory leak — use heap profiling and correlate with GC pause metrics", "Too many indexes in PostgreSQL", "Insufficient CPU for the workload"},
		CorrectAnswers: []int{1},
		Rationale:      "Gradual memory growth correlated with errors suggests a leak. Heap profiling identifies the allocation site.",
	},
	// --- Additional scenario-based (21-100) ---
	{
		Mode: "fff", Topic: "caching", DifficultyTier: "staff",
		Prompt:         "A product inventory system uses cache-aside. A DB update for a hot product triggers 10K cache invalidations. What pattern prevents thundering herd?",
		Options:        []string{"Probabilistic early expiration with jitter", "Synchronous invalidation broadcast", "Increase TTL to 24 hours", "Disable cache for hot products"},
		CorrectAnswers: []int{0},
		Rationale:      "Probabilistic early expiration spreads cache misses over time, preventing synchronized stampede when many clients miss simultaneously.",
	},
	{
		Mode: "fff", Topic: "queues", DifficultyTier: "staff",
		Prompt:         "A Kafka consumer processes payments. You need exactly-once semantics. What is the recommended approach?",
		Options:        []string{"Transactional outbox with idempotent consumer", "Two-phase commit across Kafka and DB", "Process each message exactly once via single DB transaction", "Use Kafka Streams with exactly-once config"},
		CorrectAnswers: []int{0},
		Rationale:      "Transactional outbox ensures the producer writes to Kafka only after DB commit. Idempotent consumer handles duplicates safely.",
	},
	{
		Mode: "fff", Topic: "storage", DifficultyTier: "staff",
		Prompt:         "A ride-hailing app stores millions of location updates per minute. Queries need recent positions for nearby drivers. Best storage choice?",
		Options:        []string{"PostgreSQL with spatial index", "Cassandra with time-bucketed partitions", "Redis with geospatial commands and TTL", "Elasticsearch for full-text search"},
		CorrectAnswers: []int{2},
		Rationale:      "Redis geospatial commands (GEORADIUS) with TTL suit high-write, low-retention, proximity-query workloads.",
	},
	{
		Mode: "fff", Topic: "rate-limiting", DifficultyTier: "staff",
		Prompt:         "You run 50 API gateway instances. A user must be limited to 1000 req/min globally. How do you enforce this?",
		Options:        []string{"Local in-memory counter per instance", "Redis INCR with sliding window and distributed key", "Database row per user updated on each request", "Load balancer connection limit"},
		CorrectAnswers: []int{1},
		Rationale:      "Redis provides a shared, atomic counter. Sliding window gives accurate global rate across all instances.",
	},
	{
		Mode: "fff", Topic: "observability", DifficultyTier: "senior",
		Prompt:         "A microservice calls 5 downstream services. P99 latency is high but you cannot tell which call is slow. What do you add?",
		Options:        []string{"More log statements", "Distributed tracing with span hierarchy", "APM agent with CPU profiling", "Alert on error rate only"},
		CorrectAnswers: []int{1},
		Rationale:      "Distributed tracing creates a span per call and propagates context. The trace shows which downstream contributes most to latency.",
	},
	{
		Mode: "fff", Topic: "caching", DifficultyTier: "junior",
		Prompt:         "Your API serves user-specific data that changes every few minutes. Which caching strategy fits?",
		Options:        []string{"Long TTL (1 hour) for all users", "Short TTL (30s) with cache-aside per user", "No cache — always fresh", "Cache at CDN edge with 5 min TTL"},
		CorrectAnswers: []int{1},
		Rationale:      "Short TTL with cache-aside balances freshness and load. Per-user keys avoid cross-user staleness.",
	},
	{
		Mode: "fff", Topic: "queues", DifficultyTier: "junior",
		Prompt:         "You need to process 1M image uploads. Each takes 2 seconds. How do you avoid blocking the API?",
		Options:        []string{"Process synchronously in the request handler", "Push to a queue and process asynchronously", "Store in DB and poll with cron", "Reject requests when queue is full"},
		CorrectAnswers: []int{1},
		Rationale:      "Async processing via queue decouples upload acceptance from processing, allowing fast API response and scalable workers.",
	},
	{
		Mode: "fff", Topic: "storage", DifficultyTier: "junior",
		Prompt:         "You store session data for 10M active users. Sessions expire after 24 hours. Best storage?",
		Options:        []string{"PostgreSQL with daily partition", "Redis with TTL", "DynamoDB with TTL", "In-memory hash map"},
		CorrectAnswers: []int{1},
		Rationale:      "Redis with TTL provides fast lookups and automatic expiration. Scales horizontally for session workloads.",
	},
	{
		Mode: "fff", Topic: "rate-limiting", DifficultyTier: "junior",
		Prompt:         "A fixed-window rate limiter allows 100 req/min. At 0:59 a user sends 100 requests, at 1:00 another 100. What happened?",
		Options:        []string{"User got 100 requests total", "User got 200 requests in 1 second (burst)", "Limiter blocked all requests", "Limiter reset incorrectly"},
		CorrectAnswers: []int{1},
		Rationale:      "Fixed window resets at boundary. User can double the intended rate by bursting at window edges.",
	},
	{
		Mode: "fff", Topic: "observability", DifficultyTier: "junior",
		Prompt:         "Your service restarts and error rate spikes for 30 seconds. What metric helps you correlate?",
		Options:        []string{"Total requests in last hour", "Error rate and restart events timeline", "Average response time", "Memory usage"},
		CorrectAnswers: []int{1},
		Rationale:      "Correlating error rate with restart events shows whether restarts caused the spike (e.g. connection pool drain).",
	},
	{
		Mode: "fff", Topic: "caching", DifficultyTier: "senior",
		Prompt:         "A write-through cache and DB can diverge if the cache write succeeds but DB write fails. How do you prevent inconsistency?",
		Options:        []string{"Write to DB first, then cache", "Use a distributed transaction", "Accept eventual consistency", "Write to both in parallel"},
		CorrectAnswers: []int{0},
		Rationale:      "Write to DB first ensures source of truth. Cache update after DB commit keeps cache consistent. On cache failure, next read repopulates.",
	},
	{
		Mode: "fff", Topic: "queues", DifficultyTier: "senior",
		Prompt:         "Kafka consumer lag grows during a deployment. Messages are reprocessed after restart. How do you avoid duplicate side effects?",
		Options:        []string{"Use idempotency keys and deduplicate in the consumer", "Reduce consumer count during deploy", "Increase retention to 7 days", "Process messages in batches"},
		CorrectAnswers: []int{0},
		Rationale:      "Idempotency keys let the consumer detect and skip duplicate processing. Essential for payment/order systems.",
	},
	{
		Mode: "fff", Topic: "storage", DifficultyTier: "senior",
		Prompt:         "A payment ledger must support high throughput and strong consistency. Which approach?",
		Options:        []string{"Event sourcing with single-writer partition", "CRUD with read replicas", "Append-only log with eventual consistency", "Sharded DB with no replication"},
		CorrectAnswers: []int{0},
		Rationale:      "Event sourcing with single-writer per partition provides strong consistency and audit trail. Append-only simplifies replication.",
	},
	{
		Mode: "fff", Topic: "rate-limiting", DifficultyTier: "senior",
		Prompt:         "You rate limit by API key. Some keys are shared by multiple clients. How do you avoid one bad client blocking others?",
		Options:        []string{"Rate limit per IP instead", "Use a hierarchical key (org:api_key:client_id)", "Increase limit for shared keys", "Block shared keys entirely"},
		CorrectAnswers: []int{1},
		Rationale:      "Hierarchical keys allow org-level and per-client limits. One abusive client does not exhaust the org quota.",
	},
	{
		Mode: "fff", Topic: "observability", DifficultyTier: "senior",
		Prompt:         "You deploy a new service version. Error rate increases but only for a subset of requests. How do you isolate the cause?",
		Options:        []string{"Check deployment logs only", "Segment metrics by version and request attributes", "Roll back immediately", "Increase log level globally"},
		CorrectAnswers: []int{1},
		Rationale:      "Segmenting by version and request attributes (e.g. user tier, region) isolates which subset is affected.",
	},
}

func moreCuratedQuestions() []questionSeed {
	// Additional questions to reach ~100. Topics: caching, queues, storage, rate-limiting, observability.
	seeds := []questionSeed{
		{Mode: "fff", Topic: "caching", DifficultyTier: "junior", Prompt: "A CDN caches static assets. A critical CSS file is updated. How do you invalidate?", Options: []string{"Purge by URL path", "Use versioned URLs (cache busting)", "Reduce TTL to 1 second", "Disable CDN for CSS"}, CorrectAnswers: []int{1}, Rationale: "Versioned URLs (e.g. main.abc123.css) make each version a distinct cache key. No invalidation needed."},
		{Mode: "fff", Topic: "queues", DifficultyTier: "junior", Prompt: "A producer sends 10K messages/sec. Consumer processes 2K/sec. What happens?", Options: []string{"Queue grows, consumer catches up", "Messages are dropped", "Producer blocks", "Consumer crashes"}, CorrectAnswers: []int{0}, Rationale: "Queues buffer messages. Consumer lag grows until producer slows or consumer scales."},
		{Mode: "fff", Topic: "storage", DifficultyTier: "junior", Prompt: "You need full-text search over product descriptions. Best fit?", Options: []string{"PostgreSQL LIKE", "Elasticsearch", "Redis", "CSV export"}, CorrectAnswers: []int{1}, Rationale: "Elasticsearch is built for full-text search with ranking, facets, and scale."},
		{Mode: "fff", Topic: "rate-limiting", DifficultyTier: "junior", Prompt: "Token bucket vs leaky bucket: which allows bursts?", Options: []string{"Token bucket", "Leaky bucket", "Both", "Neither"}, CorrectAnswers: []int{0}, Rationale: "Token bucket allows bursts up to bucket size. Leaky bucket smooths to constant rate."},
		{Mode: "fff", Topic: "observability", DifficultyTier: "junior", Prompt: "What is a service-level objective (SLO)?", Options: []string{"A target for reliability (e.g. 99.9% uptime)", "A deployment pipeline", "A load test result", "A database index"}, CorrectAnswers: []int{0}, Rationale: "SLOs define target reliability. SLIs measure actual performance. Error budgets drive prioritization."},
		{Mode: "fff", Topic: "caching", DifficultyTier: "senior", Prompt: "Multi-region deployment: how do you keep caches consistent?", Options: []string{"Sync all regions on every write", "Accept eventual consistency with TTL", "Single global cache", "No cache in multi-region"}, CorrectAnswers: []int{1}, Rationale: "Eventual consistency with TTL is practical. Strong consistency across regions is expensive and complex."},
		{Mode: "fff", Topic: "queues", DifficultyTier: "senior", Prompt: "Dead letter queue (DLQ): when to use?", Options: []string{"For all failed messages", "For messages that fail after max retries", "For high-priority messages", "Never"}, CorrectAnswers: []int{1}, Rationale: "DLQ holds messages that cannot be processed after retries. Enables inspection and manual replay."},
		{Mode: "fff", Topic: "storage", DifficultyTier: "senior", Prompt: "When to use CQRS (Command Query Responsibility Segregation)?", Options: []string{"When read and write patterns differ significantly", "For simple CRUD", "To reduce storage cost", "For real-time analytics only"}, CorrectAnswers: []int{0}, Rationale: "CQRS separates read and write models. Use when read queries are complex or different from write shape."},
		{Mode: "fff", Topic: "rate-limiting", DifficultyTier: "senior", Prompt: "Sliding window vs fixed window: which is more accurate?", Options: []string{"Sliding window", "Fixed window", "Same accuracy", "Depends on traffic"}, CorrectAnswers: []int{0}, Rationale: "Sliding window counts requests in a rolling window. Avoids burst at boundaries that fixed window allows."},
		{Mode: "fff", Topic: "observability", DifficultyTier: "senior", Prompt: "What does RED (Rate, Errors, Duration) measure?", Options: []string{"API request metrics", "Database size", "Cache hit rate", "Deployment frequency"}, CorrectAnswers: []int{0}, Rationale: "RED is for request-driven services: rate of requests, error rate, duration (latency)."},
		{Mode: "fff", Topic: "caching", DifficultyTier: "staff", Prompt: "Cache stampede: 10K requests miss simultaneously. Mitigation?", Options: []string{"Lock per key so only one recomputes", "Random backoff before recompute", "Pre-warm cache before TTL", "All of the above"}, CorrectAnswers: []int{3}, Rationale: "Combining lock (or coalescing), backoff, and pre-warm prevents thundering herd."},
		{Mode: "fff", Topic: "queues", DifficultyTier: "staff", Prompt: "Ordering guarantee in Kafka: how is it achieved?", Options: []string{"Per partition", "Global", "Per topic", "No ordering"}, CorrectAnswers: []int{0}, Rationale: "Kafka guarantees order only within a partition. Partition key determines which partition gets the message."},
		{Mode: "fff", Topic: "storage", DifficultyTier: "staff", Prompt: "CAP theorem: in a partition, you must choose between?", Options: []string{"Consistency and availability", "Consistency and partition tolerance", "Availability and partition tolerance", "None"}, CorrectAnswers: []int{0}, Rationale: "During partition, you cannot have both consistency and availability. You choose one."},
		{Mode: "fff", Topic: "rate-limiting", DifficultyTier: "staff", Prompt: "DDoS: rate limiting alone is insufficient because?", Options: []string{"Attackers use many IPs", "Legitimate traffic can be high", "Rate limit is per user", "All of the above"}, CorrectAnswers: []int{0}, Rationale: "Distributed attacks use many IPs. Need combined approach: rate limit, geo-blocking, WAF, capacity."},
		{Mode: "fff", Topic: "observability", DifficultyTier: "staff", Prompt: "Distributed tracing: what is a trace?", Options: []string{"A single request's path across services", "A log aggregation", "A metric dashboard", "A deployment"}, CorrectAnswers: []int{0}, Rationale: "A trace is the full path of one request. Spans represent work in each service. Trace ID links them."},
		{Mode: "fff", Topic: "caching", DifficultyTier: "junior", Prompt: "What is cache warming?", Options: []string{"Preloading cache before traffic spike", "Increasing cache temperature", "Deleting old keys", "Sharding cache"}, CorrectAnswers: []int{0}, Rationale: "Cache warming preloads frequently accessed data before high traffic to avoid cold-start misses."},
		{Mode: "fff", Topic: "queues", DifficultyTier: "junior", Prompt: "What is backpressure in a message queue?", Options: []string{"Slowing producers when consumers are overwhelmed", "Increasing queue size", "Dropping messages", "Restarting consumers"}, CorrectAnswers: []int{0}, Rationale: "Backpressure signals producers to slow down when the system cannot keep up, preventing unbounded queue growth."},
		{Mode: "fff", Topic: "storage", DifficultyTier: "junior", Prompt: "When would you use a blob store over a database?", Options: []string{"Large binary files (images, videos)", "Transactional records", "Indexed search", "Real-time analytics"}, CorrectAnswers: []int{0}, Rationale: "Blob stores (S3, GCS) are optimized for large, immutable objects. Databases are for structured, queryable data."},
		{Mode: "fff", Topic: "rate-limiting", DifficultyTier: "junior", Prompt: "What does 429 Too Many Requests indicate?", Options: []string{"Rate limit exceeded", "Server overload", "Authentication failed", "Resource not found"}, CorrectAnswers: []int{0}, Rationale: "HTTP 429 is the standard status for rate limiting. Clients should respect Retry-After header."},
		{Mode: "fff", Topic: "observability", DifficultyTier: "junior", Prompt: "What is a metric cardinality explosion?", Options: []string{"Too many unique label combinations", "Too many services", "High request volume", "Large log files"}, CorrectAnswers: []int{0}, Rationale: "High-cardinality labels (e.g. user_id) create millions of series, overwhelming metrics systems."},
		{Mode: "fff", Topic: "caching", DifficultyTier: "senior", Prompt: "Cache penetration: what is it?", Options: []string{"Requests for non-existent keys hitting DB", "Cache eviction", "Cache hit rate", "Cache replication"}, CorrectAnswers: []int{0}, Rationale: "Cache penetration occurs when queries for missing data bypass cache and hit DB. Use bloom filter or cache empty results."},
		{Mode: "fff", Topic: "queues", DifficultyTier: "senior", Prompt: "Priority queue: when to use?", Options: []string{"When some messages are more urgent", "For FIFO only", "To reduce latency", "Never"}, CorrectAnswers: []int{0}, Rationale: "Priority queues let high-priority messages (e.g. paid users) get processed before low-priority ones."},
		{Mode: "fff", Topic: "storage", DifficultyTier: "senior", Prompt: "What is a write-ahead log (WAL)?", Options: []string{"Log of changes before applying to storage", "Audit log for compliance", "Application log", "Error log"}, CorrectAnswers: []int{0}, Rationale: "WAL ensures durability: changes are logged before applied. Enables crash recovery and replication."},
		{Mode: "fff", Topic: "rate-limiting", DifficultyTier: "senior", Prompt: "Concurrent rate limit: 10 req/sec. How to enforce across 20 servers?", Options: []string{"Shared Redis counter with atomic ops", "Local counter per server", "Database row lock", "Load balancer only"}, CorrectAnswers: []int{0}, Rationale: "Redis provides atomic INCR/DECR. Sliding window or token bucket implemented in Redis works across all servers."},
		{Mode: "fff", Topic: "observability", DifficultyTier: "senior", Prompt: "Golden signals for a database?", Options: []string{"Latency, throughput, errors, saturation", "CPU, memory, disk", "Query count only", "Connection count only"}, CorrectAnswers: []int{0}, Rationale: "Latency, throughput, errors, saturation (utilization) are the four golden signals for any system."},
		{Mode: "fff", Topic: "caching", DifficultyTier: "staff", Prompt: "Multi-level cache (L1 + L2): when does it help?", Options: []string{"When L1 is local and L2 is shared", "When both are the same size", "When L1 is slower than L2", "Never"}, CorrectAnswers: []int{0}, Rationale: "L1 (local, fast) reduces L2 (shared, slower) load. Hot data in L1, warm in L2, cold in DB."},
		{Mode: "fff", Topic: "queues", DifficultyTier: "staff", Prompt: "Exactly-once processing: what makes it hard?", Options: []string{"Coordinating producer, broker, and consumer", "Network latency", "Message size", "Partition count"}, CorrectAnswers: []int{0}, Rationale: "Exactly-once requires idempotent producer, transactional consumer, and exactly-once broker semantics. All must align."},
		{Mode: "fff", Topic: "storage", DifficultyTier: "staff", Prompt: "Paxos/Raft: what do they provide?", Options: []string{"Consensus for replicated state", "Sharding", "Caching", "Full-text search"}, CorrectAnswers: []int{0}, Rationale: "Paxos and Raft are consensus algorithms. They ensure replicas agree on the same state despite failures."},
		{Mode: "fff", Topic: "rate-limiting", DifficultyTier: "staff", Prompt: "Adaptive rate limiting: how does it work?", Options: []string{"Adjust limits based on system load", "Fixed limit always", "Per-user only", "No such thing"}, CorrectAnswers: []int{0}, Rationale: "Adaptive limiters reduce limits when the system is stressed and increase when capacity is available."},
		{Mode: "fff", Topic: "observability", DifficultyTier: "staff", Prompt: "SRE error budget: what is it?", Options: []string{"Allowed unreliability before release freeze", "Budget for monitoring tools", "Incident response budget", "Deployment quota"}, CorrectAnswers: []int{0}, Rationale: "Error budget = 1 - SLO. When exhausted, focus on reliability over new features."},
		{Mode: "fff", Topic: "caching", DifficultyTier: "junior", Prompt: "LRU eviction: what does it remove first?", Options: []string{"Least recently used items", "Largest items", "Oldest by insert time", "Random items"}, CorrectAnswers: []int{0}, Rationale: "LRU evicts least recently used items when cache is full. Good for temporal locality."},
		{Mode: "fff", Topic: "queues", DifficultyTier: "junior", Prompt: "Pub/sub vs message queue: main difference?", Options: []string{"Pub/sub: fan-out to many; queue: one consumer per message", "Same thing", "Pub/sub is faster", "Queue has ordering"}, CorrectAnswers: []int{0}, Rationale: "Pub/sub broadcasts to all subscribers. Queues deliver each message to one consumer. Different use cases."},
		{Mode: "fff", Topic: "storage", DifficultyTier: "junior", Prompt: "Index on a database column: trade-off?", Options: []string{"Faster reads, slower writes", "Faster writes, slower reads", "No trade-off", "More storage only"}, CorrectAnswers: []int{0}, Rationale: "Indexes speed reads but add write overhead (index maintenance) and storage."},
		{Mode: "fff", Topic: "rate-limiting", DifficultyTier: "junior", Prompt: "Rate limit by IP: drawback?", Options: []string{"NAT/shared IP blocks many users", "Too slow", "Not accurate", "Cannot implement"}, CorrectAnswers: []int{0}, Rationale: "Many users behind NAT share one IP. Legitimate users can be blocked when one abuses."},
		{Mode: "fff", Topic: "observability", DifficultyTier: "junior", Prompt: "Log levels: when to use ERROR?", Options: []string{"When request fails and user is impacted", "For all failures", "Never", "For debug only"}, CorrectAnswers: []int{0}, Rationale: "ERROR = actionable failure affecting users. WARN = potential issue. INFO = normal operations."},
		{Mode: "fff", Topic: "caching", DifficultyTier: "senior", Prompt: "Cache coherence in distributed system?", Options: []string{"Keeping replicas consistent", "Cache size", "Eviction policy", "TTL"}, CorrectAnswers: []int{0}, Rationale: "Cache coherence ensures all replicas see consistent data. Invalidation protocols (MESI, etc.) achieve this."},
		{Mode: "fff", Topic: "queues", DifficultyTier: "senior", Prompt: "Poison pill message: what is it?", Options: []string{"Message that causes consumer to fail repeatedly", "High-priority message", "Large message", "Encrypted message"}, CorrectAnswers: []int{0}, Rationale: "Poison pills cause infinite retries. Send to DLQ after max retries to unblock the queue."},
		{Mode: "fff", Topic: "storage", DifficultyTier: "senior", Prompt: "Read-your-writes consistency: what does it mean?", Options: []string{"User sees their own writes immediately", "All users see same data", "No stale reads", "Strong consistency"}, CorrectAnswers: []int{0}, Rationale: "Read-your-writes: after a write, the same user's read returns that write. Common in session stores."},
		{Mode: "fff", Topic: "rate-limiting", DifficultyTier: "senior", Prompt: "Burst allowance in token bucket?", Options: []string{"Bucket capacity allows short bursts", "No bursts", "Unlimited bursts", "Random bursts"}, CorrectAnswers: []int{0}, Rationale: "Token bucket capacity = max burst. Refill rate = sustained rate. Allows controlled spikes."},
		{Mode: "fff", Topic: "observability", DifficultyTier: "senior", Prompt: "Alert fatigue: how to reduce?", Options: []string{"Fewer, higher-signal alerts with runbooks", "More alerts", "Longer windows", "Ignore alerts"}, CorrectAnswers: []int{0}, Rationale: "Focus on symptoms that need action. Add context and runbooks. Avoid alerting on every minor blip."},
		{Mode: "fff", Topic: "caching", DifficultyTier: "staff", Prompt: "Write-behind cache: risk?", Options: []string{"Data loss if cache fails before flush", "Slower reads", "Higher memory", "No risk"}, CorrectAnswers: []int{0}, Rationale: "Writes go to cache first, then async to DB. If cache crashes before flush, data is lost."},
		{Mode: "fff", Topic: "queues", DifficultyTier: "staff", Prompt: "Kafka consumer group: what does it provide?", Options: []string{"Load balancing and parallelism", "Ordering across partitions", "Exactly-once", "Priority"}, CorrectAnswers: []int{0}, Rationale: "Consumer group partitions workload across members. Each partition has one consumer. Enables horizontal scaling."},
		{Mode: "fff", Topic: "storage", DifficultyTier: "staff", Prompt: "CRDTs: what are they for?", Options: []string{"Conflict-free replicated data types", "Compression", "Encryption", "Indexing"}, CorrectAnswers: []int{0}, Rationale: "CRDTs allow concurrent updates without coordination. Merge automatically. Used in collaborative editing."},
		{Mode: "fff", Topic: "rate-limiting", DifficultyTier: "staff", Prompt: "Rate limit bypass via distributed requests?", Options: []string{"Use consistent hashing to same limiter", "No way to prevent", "Block all traffic", "Increase limit"}, CorrectAnswers: []int{0}, Rationale: "Consistent hashing ensures same client hits same limiter instance. Prevents spreading load to bypass."},
		{Mode: "fff", Topic: "observability", DifficultyTier: "staff", Prompt: "Distributed tracing sampling: why sample?", Options: []string{"Reduce cost and storage", "Improve accuracy", "Faster queries", "Security"}, CorrectAnswers: []int{0}, Rationale: "Full tracing is expensive. Sample (e.g. 1%) or use tail-based sampling for errors. Balance cost vs insight."},
	}
	return seeds
}

func seedQuestions(ctx context.Context, pool *pgxpool.Pool, total int) []questionSeed {
	allCurated := append(curated, moreCuratedQuestions()...)
	topics := []string{"caching", "queues", "storage", "rate-limiting", "observability"}
	tiers := []string{"junior", "senior", "staff"}
	questions := make([]questionSeed, 0, total)
	for idx := 0; idx < total; idx++ {
		var question questionSeed
		if idx < len(allCurated) {
			question = allCurated[idx]
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
