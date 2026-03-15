package room

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/google/uuid"

	"github.com/radhakrishna/archbattle/core/internal/domain/match"
	"github.com/radhakrishna/archbattle/core/internal/domain/shared"
)

const codeLen = 6
const codeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous chars

type MatchCreator interface {
	CreateMatch(ctx context.Context, req match.CreateMatchRequest) (*match.Match, error)
}

type PlayerAdder interface {
	AddPlayerToMatch(ctx context.Context, matchID, userID uuid.UUID, username string) error
}

type RoomStatusReader interface {
	GetRoomStatus(ctx context.Context, matchID uuid.UUID) (playerCount int, status string, err error)
}

type LoopRegistrar interface {
	SetExpectedPlayers(matchID uuid.UUID, count int)
	SetRoomOwner(matchID, ownerID uuid.UUID)
}

type Service struct {
	store   RoomStore
	matches MatchCreator
	adder   PlayerAdder
	status  RoomStatusReader
	loop    LoopRegistrar
}

func NewService(store RoomStore, matches MatchCreator, adder PlayerAdder, status RoomStatusReader, loop LoopRegistrar) *Service {
	return &Service{store: store, matches: matches, adder: adder, status: status, loop: loop}
}

func (s *Service) CreateRoom(ctx context.Context, userID uuid.UUID, username string) (code string, matchID uuid.UUID, err error) {
	code = generateRoomCode()
	players := []match.PlayerProfile{
		{UserID: userID, Username: username, CurrentELO: 1000, MatchesPlayed: 0},
	}
	created, err := s.matches.CreateMatch(ctx, match.CreateMatchRequest{
		Mode:    shared.ModeFFF,
		Topic:   pickRandomTopic(),
		Tier:    shared.TierSenior,
		Players: players,
	})
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("create match: %w", err)
	}
	if err := s.store.Set(ctx, code, created.ID, userID); err != nil {
		return "", uuid.Nil, fmt.Errorf("store room code: %w", err)
	}
	if s.loop != nil {
		s.loop.SetExpectedPlayers(created.ID, 2)
		s.loop.SetRoomOwner(created.ID, userID)
	}
	return code, created.ID, nil
}

func (s *Service) JoinRoom(ctx context.Context, code string, userID uuid.UUID, username string) (matchID uuid.UUID, err error) {
	matchID, _, err = s.store.Get(ctx, code)
	if err != nil {
		return uuid.Nil, fmt.Errorf("lookup room: %w", err)
	}
	if matchID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("room not found")
	}
	if err := s.adder.AddPlayerToMatch(ctx, matchID, userID, username); err != nil {
		return uuid.Nil, fmt.Errorf("add player to match: %w", err)
	}
	return matchID, nil
}

func (s *Service) GetRoom(ctx context.Context, code string) (matchID uuid.UUID, err error) {
	matchID, _, err = s.store.Get(ctx, code)
	return matchID, err
}

func (s *Service) GetRoomStatus(ctx context.Context, code string) (*RoomStatus, error) {
	matchID, _, err := s.store.Get(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("lookup room: %w", err)
	}
	if matchID == uuid.Nil {
		return nil, fmt.Errorf("room not found")
	}
	playerCount, status, err := s.status.GetRoomStatus(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("get room status: %w", err)
	}
	return &RoomStatus{MatchID: matchID, PlayerCount: playerCount, Status: status}, nil
}

func generateRoomCode() string {
	b := make([]byte, codeLen)
	max := big.NewInt(int64(len(codeChars)))
	for i := range b {
		n, _ := rand.Int(rand.Reader, max)
		b[i] = codeChars[n.Int64()]
	}
	return string(b)
}

func pickRandomTopic() shared.Topic {
	topics := []shared.Topic{
		shared.Topic("caching"),
		shared.Topic("queues"),
		shared.Topic("storage"),
		shared.Topic("rate-limiting"),
		shared.Topic("observability"),
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(topics))))
	return topics[n.Int64()]
}
