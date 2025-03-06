package blackjack

import (
	"github.com/fadedpez/tucoramirez/internal/discord"
	"github.com/fadedpez/tucoramirez/internal/games"
	"github.com/fadedpez/tucoramirez/internal/logging"
	"github.com/fadedpez/tucoramirez/pkg/games/common"
	"github.com/fadedpez/tucoramirez/pkg/storage"
)

// Factory creates blackjack games and managers
type Factory struct {
	session discord.SessionHandler
	storage storage.Storage
}

// NewFactory creates a new blackjack factory
func NewFactory(session discord.SessionHandler, storage storage.Storage) *Factory {
	return &Factory{
		session: session,
		storage: storage,
	}
}

// CreateGame creates a new blackjack game
func (f *Factory) CreateGame(creatorID, channelID string, players []string) games.Game {
	game := NewGame(creatorID, channelID)
	for _, playerID := range players {
		player := common.NewPlayer(playerID)
		if err := game.AddPlayer(player); err != nil {
			logging.Default.Error("Failed to add player to game: %v", err)
		}
	}
	return game
}

// CreateManager creates a new blackjack game manager
func (f *Factory) CreateManager() games.Manager {
	return NewManager(f.storage)
}
