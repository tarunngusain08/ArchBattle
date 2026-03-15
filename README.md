# ArchBattle

**The System Design Knowledge Commons** — compete, learn, discuss.

ArchBattle is a real-time competitive system design learning platform where engineers compete in multiplayer battles, build daily learning habits through a shared challenge, and contribute to a community knowledge base via structured discussion threads.

---

## Architecture Overview

ArchBattle Phase 1 runs three deployable units:

| Service | Role |
|---------|------|
| **Core Service** | Go monolith: matchmaking, game engine, question engine, leaderboard, daily challenge, WebSocket gateway. Uses Redis Streams for event transport and Postgres for persistence. |
| **AI Service** | Stateless Go service wrapping Claude API: tutor, question drafting, learning summary, discussion summary. Separate for cost metering and rate limiting. |
| **Frontend** | React + TypeScript SPA served via CDN. Communicates with Core via REST + WebSocket. |

```
Browser (React + WS)  <--->  Core Service (Go)  <--->  AI Service (Go)  <--->  Claude API
                                    |
                                    v
                            Redis + Postgres
```

- **Event transport**: Redis Streams per match; reconnect replays from last sequence.
- **Answer ordering**: ZADD NX + monotonic counter for deterministic sub-millisecond tie-breaking.
- **ELO**: Per-tier (Junior/Senior/Staff), performance-weighted delta.

---

## Local Development Setup

### Prerequisites

- Go 1.24+
- Node.js 22+
- Docker & Docker Compose
- (Optional) Anthropic API key for AI features

### 1. Start infrastructure

```bash
docker compose -f deployments/docker-compose.yml up -d redis postgres
```

### 2. Configure environment

```bash
cp .env.example .env
# Edit .env: set JWT_SECRET, AI_INTERNAL_SECRET, ANTHROPIC_API_KEY (if using AI)
```

### 3. Run migrations

```bash
cd core && go run ./cmd/server migrate
```

### 4. Seed development data (optional)

```bash
cd scripts/seed && go run . --questions 20 --daily 7
```

### 5. Start services

**Terminal 1 – Core:**
```bash
cd core && go run ./cmd/server
```

**Terminal 2 – AI Service:**
```bash
cd ai-service && go run ./cmd/server
```

**Terminal 3 – Frontend:**
```bash
cd frontend && npm run dev
```

Open http://localhost:5173. The frontend proxies API requests to the Core service.

### Full stack via Docker

```bash
docker compose -f deployments/docker-compose.yml up --build -d
```

Access at http://localhost (nginx reverse proxy).

---

## API Summary

### REST Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | No | Register with email + password |
| POST | `/auth/login` | No | Login, returns JWT |
| POST | `/auth/logout` | Bearer | Invalidate session |
| GET | `/users/me` | Bearer | Current user profile |
| POST | `/match/queue` | Bearer | Join matchmaking queue |
| DELETE | `/match/queue` | Bearer | Leave queue |
| GET | `/daily-challenge` | Bearer | Today's daily challenge |
| POST | `/daily-submit` | Bearer | Submit daily answers |
| GET | `/daily-share-card` | Bearer | Share card text |
| GET/POST | `/daily-challenge/{date}/discussion/` | Bearer | List/create discussion entries |
| POST | `/daily-challenge/{date}/discussion/{id}/upvote` | Bearer | Upvote entry |
| GET | `/leaderboard` | Bearer | Global/weekly leaderboard |
| POST | `/questions/{id}/dispute` | Bearer | Flag question for review |
| POST | `/api/tutor` | Bearer | AI tutor (post-match) |

### WebSocket (`/ws`)

- **Client → Server**: `join_match`, `answer_submit`, `reconnect`, `cross_match_accept`, `accept_solo`, `rematch_request`, `ping`
- **Server → Client**: `match_found`, `lobby_state`, `question_broadcast`, `score_update`, `question_reveal`, `match_end`, `learning_summary`, `solo_fallback_offer`, `cross_match_prompt`, `reconnect_complete`, `pong`

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | — | Postgres connection string |
| `REDIS_URL` | Yes | — | Redis connection string |
| `JWT_SECRET` | Yes | — | Secret for JWT signing |
| `AI_SERVICE_URL` | No | http://localhost:8081 | AI service base URL |
| `AI_INTERNAL_SECRET` | No | — | Shared secret for Core ↔ AI auth |
| `ANTHROPIC_API_KEY` | AI only | — | Anthropic API key for Claude |
| `CORE_HTTP_PORT` | No | 8080 | Core HTTP port |
| `ALLOWED_ORIGINS` | No | http://localhost:5173 | CORS origins (comma-separated) |
| `STREAK_GRACE_HOURS` | No | 48 | Streak grace period (Phase 1) |
| `FREE_MATCH_LIMIT_PER_DAY` | No | 5 | Free-tier match limit |
| `FREE_TUTOR_LIMIT_PER_DAY` | No | 3 | Free-tier tutor sessions per match |

See `.env.example` for the full list.

---

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make deps` | Install all dependencies |
| `make build` | Build core, ai-service, frontend |
| `make test` | Run Go tests |
| `make lint` | Run linters |
| `make compose-up` | Start full stack with Docker |
| `make compose-down` | Stop and remove containers |
| `make migrate-core` | Run database migrations |
| `make seed` | Seed development data |

---

## Testing

```bash
# Unit tests (default)
cd core && go test ./...

# Integration tests (requires DATABASE_URL, REDIS_URL)
go test -tags=integration ./core/internal/adapter/inbound/http/...
```

---

## Documentation

- `docs/` — PRD, Design Doc, HLD/LLD diagrams, sequence diagrams
- `docs/lld_postgres_erd.html` — Postgres ERD (Mermaid)
- `docs/system_context.svg` — System context diagram
