package storage

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// MockStorage is a mock implementation of Storage
type MockStorage struct {
	mock.Mock
}

// NewMockStorage creates a new mock storage
func NewMockStorage(t mock.TestingT) *MockStorage {
	mock := &MockStorage{}
	mock.Test(t)
	return mock
}

// SaveGame mocks the SaveGame method
func (m *MockStorage) SaveGame(ctx context.Context, state *GameState) error {
	args := m.Called(ctx, state)
	return args.Error(0)
}

// LoadGame mocks the LoadGame method
func (m *MockStorage) LoadGame(ctx context.Context, id string) (*GameState, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GameState), args.Error(1)
}

// LoadGameByChannel mocks the LoadGameByChannel method
func (m *MockStorage) LoadGameByChannel(ctx context.Context, channelID string) (*GameState, error) {
	args := m.Called(ctx, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*GameState), args.Error(1)
}

// DeleteGame mocks the DeleteGame method
func (m *MockStorage) DeleteGame(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// ListGames mocks the ListGames method
func (m *MockStorage) ListGames(ctx context.Context) ([]*GameState, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*GameState), args.Error(1)
}

// CleanupOldGames mocks the CleanupOldGames method
func (m *MockStorage) CleanupOldGames(ctx context.Context, maxAge time.Duration) error {
	args := m.Called(ctx, maxAge)
	return args.Error(0)
}
