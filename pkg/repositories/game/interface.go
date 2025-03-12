package game

import (
	"context"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

// Repository defines storage operations for deck state and game results
type Repository interface {
	// Deck operations
	SaveDeck(ctx context.Context, channelID string, deck []*entities.Card) error
	GetDeck(ctx context.Context, channelID string) ([]*entities.Card, error)

	// Game results
	SaveGameResult(ctx context.Context, result *entities.GameResult) error
	GetPlayerResults(ctx context.Context, playerID string) ([]*entities.GameResult, error)
	GetChannelResults(ctx context.Context, channelID string, limit int) ([]*entities.GameResult, error)
}
