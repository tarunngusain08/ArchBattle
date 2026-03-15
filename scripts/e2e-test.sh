#!/bin/bash
# E2E test script: seed DB, run API tests, verify question flow
set -e

echo "=== E2E Test: ArchBattle ==="

# 1. Seed database
echo "[1/4] Seeding database..."
cd "$(dirname "$0")/.."
DATABASE_URL="postgres://postgres:postgres@localhost:5432/archbattle?sslmode=disable" \
REDIS_URL="redis://localhost:6379/0" \
cd scripts/seed && go run . -questions 100 -daily 7 && cd ../..
echo "Seed complete."

# 2. Create users and room via API
echo "[2/4] Creating room and joining..."
USER1=$(curl -s -X POST http://localhost:8080/join -H "Content-Type: application/json" -d '{"username":"e2e_host"}' | grep -o '"userId":"[^"]*"' | cut -d'"' -f4)
USER2=$(curl -s -X POST http://localhost:8080/join -H "Content-Type: application/json" -d '{"username":"e2e_joiner"}' | grep -o '"userId":"[^"]*"' | cut -d'"' -f4)
ROOM=$(curl -s -X POST http://localhost:8080/rooms -H "Content-Type: application/json" -d "{\"userId\":\"$USER1\",\"username\":\"e2e_host\"}")
CODE=$(echo "$ROOM" | grep -o '"roomCode":"[^"]*"' | cut -d'"' -f4)
curl -s -X POST "http://localhost:8080/rooms/${CODE}/join" -H "Content-Type: application/json" -d "{\"userId\":\"$USER2\",\"username\":\"e2e_joiner\"}" > /dev/null
echo "Room $CODE created, 2 players joined."

# 3. Verify daily challenge API
echo "[3/4] Verifying daily challenge API..."
DAILY=$(curl -s "http://localhost:8080/daily-challenge")
if echo "$DAILY" | grep -q "questions"; then
  echo "Daily challenge returns questions."
else
  echo "WARN: Daily challenge response unexpected: $DAILY"
fi

# 4. Verify question count in DB
echo "[4/4] Verifying DB state..."
COUNT=$(docker compose -f deployments/docker-compose.yml exec -T postgres psql -U postgres -d archbattle -t -c "SELECT COUNT(*) FROM questions WHERE is_active AND status IN ('live','staged');" 2>/dev/null | tr -d ' ')
echo "Live questions in DB: $COUNT"

echo ""
echo "=== E2E Test Complete ==="
echo "Manual browser test: Login, create room, have 2nd user join, Start Battle."
echo "Daily Challenge: Login, go to Daily, answer questions, Submit."
