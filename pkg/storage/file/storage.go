package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/storage"
)

// Storage implements file-based storage for game states
type Storage struct {
	path    string
	mu      sync.RWMutex
	games   map[string]*storage.GameState
	options *storage.Options
}

// New creates a new file storage instance
func New(options *storage.Options) (*Storage, error) {
	if options == nil {
		options = storage.NewOptions()
	}

	s := &Storage{
		path:    options.Path,
		games:   make(map[string]*storage.GameState),
		options: options,
	}

	// Load existing games from file
	if err := s.load(); err != nil {
		return nil, fmt.Errorf("failed to load games: %w", err)
	}

	// Start cleanup goroutine if enabled
	if options.AutoCleanup {
		go s.cleanupRoutine()
	}

	return s, nil
}

// SaveGame saves or updates a game state
func (s *Storage) SaveGame(ctx context.Context, state *storage.GameState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update timestamps
	now := time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	state.UpdatedAt = now

	// Save to memory
	s.games[state.ID] = state

	// Save to file
	return s.save()
}

// LoadGame loads a game state by ID
func (s *Storage) LoadGame(ctx context.Context, id string) (*storage.GameState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	game, ok := s.games[id]
	if !ok {
		return nil, fmt.Errorf("game not found: %s", id)
	}

	return game, nil
}

// LoadGameByChannel loads the active game state for a channel
func (s *Storage) LoadGameByChannel(ctx context.Context, channelID string) (*storage.GameState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, game := range s.games {
		if game.ChannelID == channelID {
			return game, nil
		}
	}

	return nil, fmt.Errorf("no game found for channel: %s", channelID)
}

// DeleteGame deletes a game state
func (s *Storage) DeleteGame(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.games, id)
	return s.save()
}

// ListGames lists all game states
func (s *Storage) ListGames(ctx context.Context) ([]*storage.GameState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	games := make([]*storage.GameState, 0, len(s.games))
	for _, game := range s.games {
		games = append(games, game)
	}

	return games, nil
}

// CleanupOldGames removes games older than maxAge
func (s *Storage) CleanupOldGames(ctx context.Context, maxAge time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, game := range s.games {
		if now.Sub(game.UpdatedAt) > maxAge {
			delete(s.games, id)
		}
	}

	return s.save()
}

// Helper functions

func (s *Storage) load() error {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.games)
}

func (s *Storage) save() error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal and save
	data, err := json.Marshal(s.games)
	if err != nil {
		return fmt.Errorf("failed to marshal games: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func (s *Storage) cleanupRoutine() {
	ticker := time.NewTicker(s.options.MaxGameAge / 4)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.CleanupOldGames(context.Background(), s.options.MaxGameAge); err != nil {
			fmt.Printf("Error cleaning up old games: %v\n", err)
		}
	}
}
