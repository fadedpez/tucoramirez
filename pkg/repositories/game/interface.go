package game

import (
	"context"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

//go:generate mockgen -source=$GOFILE -destination=mock/mock.go -package=mock_game

// Repository defines storage operations for deck state and game results
type Repository interface {
	// Deck operations
	SaveDeck(ctx context.Context, channelID string, deck []*entities.Card) error
	GetDeck(ctx context.Context, channelID string) ([]*entities.Card, error)

	// Game results
	SaveGameResult(ctx context.Context, result *entities.GameResult) error
	GetPlayerResults(ctx context.Context, playerID string) ([]*entities.GameResult, error)
	GetChannelResults(ctx context.Context, channelID string, limit int) ([]*entities.GameResult, error)
	
	// Player statistics
	GetPlayerStatistics(ctx context.Context, playerID string, gameType entities.GameState) (*entities.PlayerStatistics, error)
	GetAllPlayerStatistics(ctx context.Context, gameType entities.GameState) ([]*entities.PlayerStatistics, error)
	UpdatePlayerStatistics(ctx context.Context, gameResult *entities.GameResult) error
	PruneGameResultsPerPlayer(ctx context.Context, maxMatchesPerPlayer int) error
	ArchiveGameResults(ctx context.Context, gameResults []*entities.GameResult) error
	
	// Elasticsearch specific operations
	IndexGameResult(ctx context.Context, gameResult *entities.GameResult) error
	GetPlayerStatisticsFromES(ctx context.Context, playerID string, gameType entities.GameState) (*entities.PlayerStatistics, error)
	GetAllPlayerStatisticsFromES(ctx context.Context, gameType entities.GameState) ([]*entities.PlayerStatistics, error)

	// Close closes any resources used by the repository
	Close() error
}
