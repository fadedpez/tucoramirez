package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// Common storage errors
var (
	ErrGameNotFound = errors.New("game not found")
)

// GameState represents the state of a game that can be stored
type GameState struct {
	ID        string          `json:"id"`
	GameType  string          `json:"game_type"`
	ChannelID string          `json:"channel_id"`
	CreatorID string          `json:"creator_id"`
	State     json.RawMessage `json:"state"`      // Game-specific state
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Storage defines the interface for game state persistence
type Storage interface {
	// SaveGame saves or updates a game state
	SaveGame(ctx context.Context, state *GameState) error

	// LoadGame loads a game state by ID
	LoadGame(ctx context.Context, id string) (*GameState, error)

	// LoadGameByChannel loads the active game state for a channel
	LoadGameByChannel(ctx context.Context, channelID string) (*GameState, error)

	// DeleteGame deletes a game state
	DeleteGame(ctx context.Context, id string) error

	// ListGames lists all game states
	ListGames(ctx context.Context) ([]*GameState, error)

	// CleanupOldGames removes games older than maxAge
	CleanupOldGames(ctx context.Context, maxAge time.Duration) error
}

// Options represents storage configuration options
type Options struct {
	Path        string
	MaxGameAge  time.Duration
	AutoCleanup bool
}

// NewOptions creates a new Options with default values
func NewOptions() *Options {
	return &Options{
		Path:        "games.db",
		MaxGameAge:  24 * time.Hour,
		AutoCleanup: true,
	}
}
