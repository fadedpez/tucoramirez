package mock

import (
	"context"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/storage"
	"github.com/stretchr/testify/mock"
)

// Storage is a mock implementation of storage.Storage
type Storage struct {
	mock.Mock
}

func New() *Storage {
	return &Storage{}
}

func (s *Storage) SaveGame(ctx context.Context, state *storage.GameState) error {
	args := s.Called(ctx, state)
	return args.Error(0)
}

func (s *Storage) LoadGame(ctx context.Context, id string) (*storage.GameState, error) {
	args := s.Called(ctx, id)
	if state, ok := args.Get(0).(*storage.GameState); ok {
		return state, args.Error(1)
	}
	return nil, args.Error(1)
}

func (s *Storage) LoadGameByChannel(ctx context.Context, channelID string) (*storage.GameState, error) {
	args := s.Called(ctx, channelID)
	if state, ok := args.Get(0).(*storage.GameState); ok {
		return state, args.Error(1)
	}
	return nil, args.Error(1)
}

func (s *Storage) DeleteGame(ctx context.Context, id string) error {
	args := s.Called(ctx, id)
	return args.Error(0)
}

func (s *Storage) ListGames(ctx context.Context) ([]*storage.GameState, error) {
	args := s.Called(ctx)
	if games, ok := args.Get(0).([]*storage.GameState); ok {
		return games, args.Error(1)
	}
	return nil, args.Error(1)
}

func (s *Storage) CleanupOldGames(ctx context.Context, maxAge time.Duration) error {
	args := s.Called(ctx, maxAge)
	return args.Error(0)
}

func (s *Storage) Close() error {
	args := s.Called()
	return args.Error(0)
}
