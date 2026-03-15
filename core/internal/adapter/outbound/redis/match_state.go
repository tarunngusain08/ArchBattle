package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	domainmatch "github.com/radhakrishna/archbattle/core/internal/domain/match"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

type MatchStateStore struct {
	client *goredis.Client
}

func NewMatchStateStore(client *goredis.Client) *MatchStateStore {
	return &MatchStateStore{client: client}
}

func (s *MatchStateStore) SetMatchState(ctx context.Context, matchID uuid.UUID, state *domainmatch.MatchStateData) error {
	playerIDs, err := json.Marshal(state.PlayerIDs)
	if err != nil {
		return fmt.Errorf("encode player ids: %w", err)
	}
	values := map[string]any{
		"state":               string(state.State),
		"mode":                string(state.Mode),
		"topic":               string(state.Topic),
		"tier":                string(state.Tier),
		"player_ids":          playerIDs,
		"question_index":      state.QuestionIndex,
		"current_question_id": state.CurrentQuestionID.String(),
		"updated_at":          state.UpdatedAt.Format(time.RFC3339Nano),
	}
	if err := s.client.HSet(ctx, matchStateKey(matchID), values).Err(); err != nil {
		return fmt.Errorf("write match state: %w", err)
	}
	_ = s.client.Expire(ctx, matchStateKey(matchID), 15*time.Minute).Err()
	_ = s.client.Expire(ctx, matchPlayersSetKey(matchID), 15*time.Minute).Err()
	return nil
}

func (s *MatchStateStore) GetMatchState(ctx context.Context, matchID uuid.UUID) (*domainmatch.MatchStateData, error) {
	values, err := s.client.HGetAll(ctx, matchStateKey(matchID)).Result()
	if err != nil {
		return nil, fmt.Errorf("read match state: %w", err)
	}
	if len(values) == 0 {
		return nil, nil
	}
	state := &domainmatch.MatchStateData{MatchID: matchID}
	state.State = domainmatch.MatchState(values["state"])
	state.Mode = shared.Mode(values["mode"])
	state.Topic = shared.Topic(values["topic"])
	state.Tier = shared.Tier(values["tier"])
	if values["current_question_id"] != "" {
		state.CurrentQuestionID, _ = uuid.Parse(values["current_question_id"])
	}
	if values["question_index"] != "" {
		state.QuestionIndex, _ = strconv.Atoi(values["question_index"])
	}
	if values["updated_at"] != "" {
		state.UpdatedAt, _ = time.Parse(time.RFC3339Nano, values["updated_at"])
	}
	if raw, ok := values["player_ids"]; ok && raw != "" {
		_ = json.Unmarshal([]byte(raw), &state.PlayerIDs)
	}
	return state, nil
}

func (s *MatchStateStore) SetPlayerStatus(ctx context.Context, matchID, userID uuid.UUID, status string) error {
	if err := s.client.HSet(ctx, matchStateKey(matchID), fmt.Sprintf("p:%s", userID), status).Err(); err != nil {
		return fmt.Errorf("set player status: %w", err)
	}
	return nil
}

func (s *MatchStateStore) GetPlayerStatus(ctx context.Context, matchID, userID uuid.UUID) (string, error) {
	value, err := s.client.HGet(ctx, matchStateKey(matchID), fmt.Sprintf("p:%s", userID)).Result()
	if err == goredis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get player status: %w", err)
	}
	return value, nil
}

// appendPlayerScript atomically adds a player to the match's player set and,
// if newly added, patches the player_ids JSON array in the match hash.
// Keys: [0] = match hash key, [1] = players set key
// Args: [0] = userID string, [1] = updated_at timestamp
var appendPlayerScript = goredis.NewScript(`
local added = redis.call("SADD", KEYS[2], ARGV[1])
if added == 0 then
  return 0
end
local raw = redis.call("HGET", KEYS[1], "player_ids")
local arr
if raw == false or raw == "" then
  arr = {}
else
  arr = cjson.decode(raw)
end
table.insert(arr, ARGV[1])
redis.call("HSET", KEYS[1], "player_ids", cjson.encode(arr), "updated_at", ARGV[2])
return 1
`)

func (s *MatchStateStore) AppendPlayer(ctx context.Context, matchID, userID uuid.UUID) error {
	matchKey := matchStateKey(matchID)
	setKey := matchPlayersSetKey(matchID)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if err := appendPlayerScript.Run(ctx, s.client, []string{matchKey, setKey}, userID.String(), now).Err(); err != nil && err != goredis.Nil {
		return fmt.Errorf("atomic append player: %w", err)
	}
	return nil
}

func (s *MatchStateStore) SetCurrentQuestion(ctx context.Context, matchID, questionID uuid.UUID, index int) error {
	if err := s.client.HSet(ctx, matchStateKey(matchID), "current_question_id", questionID.String(), "question_index", index, "updated_at", time.Now().UTC().Format(time.RFC3339Nano)).Err(); err != nil {
		return fmt.Errorf("set current question: %w", err)
	}
	return nil
}

func (s *MatchStateStore) SetExpiry(ctx context.Context, matchID uuid.UUID, ttl time.Duration) error {
	if err := s.client.Expire(ctx, matchPlayersSetKey(matchID), ttl).Err(); err != nil {
		slog.Warn("expire players set failed", "match", matchID, "error", err)
	}
	if err := s.client.Expire(ctx, matchStateKey(matchID), ttl).Err(); err != nil {
		return fmt.Errorf("expire match state: %w", err)
	}
	return nil
}

func matchStateKey(matchID uuid.UUID) string {
	return fmt.Sprintf("match:%s", matchID)
}

func matchPlayersSetKey(matchID uuid.UUID) string {
	return fmt.Sprintf("match:%s:players", matchID)
}
