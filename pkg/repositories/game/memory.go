package game

import (
	"context"
	"sync"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

// MemoryRepository implements Repository interface with in-memory storage
type MemoryRepository struct {
	mu sync.RWMutex
	// Map of channelID to deck
	decks map[string][]*entities.Card
	// Map of channelID to game results
	channelResults map[string][]*entities.GameResult
	// Map of playerID to game results
	playerResults map[string][]*entities.GameResult
}

// NewMemoryRepository creates a new in-memory repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		decks:          make(map[string][]*entities.Card),
		channelResults: make(map[string][]*entities.GameResult),
		playerResults:  make(map[string][]*entities.GameResult),
	}
}

// SaveDeck stores a deck for a channel
func (r *MemoryRepository) SaveDeck(ctx context.Context, channelID string, deck []*entities.Card) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.decks[channelID] = deck
	return nil
}

// GetDeck retrieves a deck for a channel
func (r *MemoryRepository) GetDeck(ctx context.Context, channelID string) ([]*entities.Card, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	deck, exists := r.decks[channelID]
	if !exists {
		return nil, nil // Return empty deck if none exists
	}
	return deck, nil
}

// SaveGameResult stores a game result and updates both channel and player histories
func (r *MemoryRepository) SaveGameResult(ctx context.Context, result *entities.GameResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Add to channel results
	r.channelResults[result.ChannelID] = append(r.channelResults[result.ChannelID], result)

	// Add to each player's results
	for _, pr := range result.PlayerResults {
		r.playerResults[pr.PlayerID] = append(r.playerResults[pr.PlayerID], &entities.GameResult{
			ChannelID:     result.ChannelID,
			GameType:      result.GameType,
			CompletedAt:   result.CompletedAt,
			PlayerResults: []*entities.PlayerResult{pr},
			Details:       result.Details,
		})
	}

	return nil
}

// GetPlayerResults retrieves game results for a player
func (r *MemoryRepository) GetPlayerResults(ctx context.Context, playerID string) ([]*entities.GameResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := r.playerResults[playerID]
	if results == nil {
		return []*entities.GameResult{}, nil
	}
	return results, nil
}

// GetChannelResults retrieves recent game results for a channel
func (r *MemoryRepository) GetChannelResults(ctx context.Context, channelID string, limit int) ([]*entities.GameResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := r.channelResults[channelID]
	if results == nil {
		return []*entities.GameResult{}, nil
	}

	// If we have more results than the limit, return only the most recent ones
	if len(results) > limit {
		return results[len(results)-limit:], nil
	}
	return results, nil
}

// Close is a no-op for memory repository since there are no resources to close
func (r *MemoryRepository) Close() error {
	return nil
}
