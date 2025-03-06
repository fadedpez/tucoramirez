package blackjack

import (
	"context"
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/discord"
	"github.com/fadedpez/tucoramirez/pkg/games/common"
	"github.com/fadedpez/tucoramirez/pkg/storage"
)

// Manager manages all active blackjack games
type Manager struct {
	storage storage.Storage
	games   map[string]*Game
	mu      sync.RWMutex
}

// NewManager creates a new blackjack game manager
func NewManager(storage storage.Storage) *Manager {
	if storage == nil {
		panic("storage cannot be nil")
	}
	return &Manager{
		storage: storage,
		games:   make(map[string]*Game),
	}
}

// HandleStart handles the start command for blackjack
func (m *Manager) HandleStart(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	// Validate input
	if i.Member == nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("invalid member"))
		return
	}
	if i.Member.User == nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("invalid user"))
		return
	}
	if i.Member.User.ID == "" {
		discord.SendErrorResponse(s, i, fmt.Errorf("invalid user ID"))
		return
	}
	if i.ChannelID == "" {
		discord.SendErrorResponse(s, i, fmt.Errorf("invalid channel ID"))
		return
	}

	// Check if we already have a game for this channel
	state, err := m.storage.LoadGameByChannel(context.Background(), i.ChannelID)
	if err == nil && state != nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("there is already an active game in this channel"))
		return
	}

	// Create a new game
	game := NewGame(i.Member.User.ID, i.ChannelID)

	// Add the creator as the first player
	player := common.NewPlayer(i.Member.User.ID)
	player.Username = i.Member.User.Username
	if err := game.AddPlayer(player); err != nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("failed to add player to game: %w", err))
		return
	}

	// Store the game
	if err := m.storage.SaveGame(context.Background(), &storage.GameState{
		ID:        game.ID,
		GameType:  "blackjack",
		ChannelID: i.ChannelID,
		CreatorID: i.Member.User.ID,
		State:     game.MarshalState(),
	}); err != nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("failed to save game: %w", err))
		return
	}

	// Store the game in memory
	m.AddGame(game)

	// Send the initial game message
	if err := discord.SendResponse(s, i, discord.NewResponse(game.String(), game.GetButtons())); err != nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("failed to send game message: %w", err))
		return
	}

	// Send a channel message to notify others
	if _, err := s.ChannelMessageSend(i.ChannelID, "A new blackjack game has started! Click Join to play!"); err != nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("failed to send channel message: %w", err))
		return
	}
}

// HandleButton handles button interactions for blackjack games
func (m *Manager) HandleButton(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	// Validate input
	if s == nil {
		panic("session handler cannot be nil")
	}
	if i == nil {
		panic("interaction cannot be nil")
	}
	if i.ChannelID == "" {
		discord.SendErrorResponse(s, i, fmt.Errorf("invalid channel ID"))
		return
	}
	if i.Member == nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("invalid member"))
		return
	}
	if i.Member.User == nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("invalid user"))
		return
	}
	if i.Member.User.ID == "" {
		discord.SendErrorResponse(s, i, fmt.Errorf("invalid user ID"))
		return
	}

	// Get the game from memory first
	game := m.GetGame(i.ChannelID)
	if game == nil {
		// If not in memory, try loading from storage
		state, err := m.storage.LoadGameByChannel(context.Background(), i.ChannelID)
		if err != nil {
			if err == storage.ErrGameNotFound {
				discord.SendErrorResponse(s, i, fmt.Errorf("no active game in this channel"))
				return
			}
			discord.SendErrorResponse(s, i, fmt.Errorf("failed to load game state: %w", err))
			return
		}

		// Create a new game from the state
		game = NewGame(state.CreatorID, state.ChannelID)
		game.ID = state.ID
		if err := game.LoadState(state.State); err != nil {
			discord.SendErrorResponse(s, i, fmt.Errorf("failed to unmarshal game state: %w", err))
			return
		}
		m.AddGame(game)
	}

	// Special handling for play again since it creates a new game
	if i.MessageComponentData().CustomID == "blackjack_playagain" {
		// Verify game is finished
		if !game.IsFinished() {
			discord.SendErrorResponse(s, i, fmt.Errorf("game must be finished before starting a new one"))
			return
		}

		newGame, err := game.HandlePlayAgain(s, i)
		if err != nil {
			discord.SendErrorResponse(s, i, fmt.Errorf("failed to create new game: %w", err))
			return
		}

		// Delete the old game and save the new one atomically
		if err := m.storage.DeleteGame(context.Background(), game.ID); err != nil {
			discord.SendErrorResponse(s, i, fmt.Errorf("failed to delete old game: %w", err))
			return
		}

		newState := &storage.GameState{
			ID:        newGame.ID,
			GameType:  "blackjack",
			ChannelID: newGame.ChannelID,
			CreatorID: newGame.CreatorID,
			State:     newGame.MarshalState(),
		}
		if err := m.storage.SaveGame(context.Background(), newState); err != nil {
			// Try to restore the old game since we couldn't save the new one
			if restoreErr := m.storage.SaveGame(context.Background(), &storage.GameState{
				ID:        game.ID,
				GameType:  "blackjack",
				ChannelID: game.ChannelID,
				CreatorID: game.CreatorID,
				State:     game.MarshalState(),
			}); restoreErr != nil {
				discord.SendErrorResponse(s, i, fmt.Errorf("failed to save new game and restore old game: %w", err))
				return
			}
			discord.SendErrorResponse(s, i, fmt.Errorf("failed to save new game: %w", err))
			return
		}

		// Add the new game to memory
		m.AddGame(newGame)

		// Send the new game state
		if err := discord.SendGameResponse(s, i, newGame.String(), newGame.GetButtons()); err != nil {
			discord.SendErrorResponse(s, i, fmt.Errorf("failed to send game: %w", err))
			return
		}

		// Send a channel message to notify others
		if _, err := s.ChannelMessageSend(i.ChannelID, "A new blackjack game has started! Click Join to play!"); err != nil {
			discord.SendErrorResponse(s, i, fmt.Errorf("failed to send channel message: %w", err))
			return
		}
		return
	}

	// Handle other button presses
	game.HandleButton(s, i)

	// Save the game state if it's not finished
	if !game.IsFinished() {
		if err := m.storage.SaveGame(context.Background(), &storage.GameState{
			ID:        game.ID,
			GameType:  "blackjack",
			ChannelID: game.ChannelID,
			CreatorID: game.CreatorID,
			State:     game.MarshalState(),
		}); err != nil {
			discord.SendErrorResponse(s, i, fmt.Errorf("failed to save game state: %w", err))
			return
		}
	} else {
		// Remove finished game from storage and memory
		if err := m.storage.DeleteGame(context.Background(), game.ID); err != nil {
			discord.SendErrorResponse(s, i, fmt.Errorf("failed to delete finished game: %w", err))
			return
		}
		delete(m.games, game.ChannelID)
	}
}

// AddGame adds a game to the manager
func (m *Manager) AddGame(game *Game) {
	if game == nil {
		panic("game cannot be nil")
	}
	if game.ChannelID == "" {
		panic("game channel ID cannot be empty")
	}
	if game.CreatorID == "" {
		panic("game creator ID cannot be empty")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.games[game.ChannelID] = game
}

// GetGame gets a game from the manager by channel ID
func (m *Manager) GetGame(channelID string) *Game {
	if channelID == "" {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.games[channelID]
}

// HasGame checks if a game exists in the manager by ID
func (m *Manager) HasGame(id string) bool {
	if id == "" {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, game := range m.games {
		if game.ID == id {
			return true
		}
	}
	return false
}

// CleanupFinishedGames removes all finished games from memory and storage
func (m *Manager) CleanupFinishedGames() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for channelID, game := range m.games {
		if game.IsFinished() {
			delete(m.games, channelID)
			m.storage.DeleteGame(context.Background(), game.ID)
		}
	}
}
