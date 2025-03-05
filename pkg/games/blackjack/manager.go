package blackjack

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/utils"
	"github.com/fadedpez/tucoramirez/pkg/cards"
	"github.com/fadedpez/tucoramirez/pkg/games/common"
)

// Manager handles all blackjack games
type Manager struct {
	session *discordgo.Session
	games   map[string]*Game // key is channelID
	mu      sync.RWMutex
}

// NewManager creates a new blackjack game manager
func NewManager(session *discordgo.Session) *Manager {
	return &Manager{
		session: session,
		games:   make(map[string]*Game),
	}
}

// HandleStart handles the start of a new blackjack game
func (m *Manager) HandleStart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID

	// Check for existing game
	m.mu.RLock()
	game, exists := m.games[channelID]
	m.mu.RUnlock()

	if exists && !game.IsFinished() {
		utils.SendErrorResponse(s, i, utils.NewGameError(utils.ErrGameInProgress, "A game is already in progress in this channel!"))
		return
	}

	// Create new game
	game = NewGame(i.Member.User.ID, channelID)
	
	// Add creator as first player
	player := &common.Player{
		ID:       i.Member.User.ID,
		Username: i.Member.User.Username,
	}
	game.AddPlayer(player)

	// Store game
	m.mu.Lock()
	m.games[channelID] = game
	m.mu.Unlock()

	// Create buttons for joining and starting
	buttons := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Join Game",
					Style:    discordgo.SuccessButton,
					CustomID: "blackjack_join",
				},
				discordgo.Button{
					Label:    "Start Game",
					Style:    discordgo.PrimaryButton,
					CustomID: "blackjack_start",
				},
			},
		},
	}

	// Send initial message
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    fmt.Sprintf("🎲 Blackjack Game (1/%d players)\nPlayers: %s", MaxPlayers, player.Username),
			Components: buttons,
		},
	})
	if err != nil {
		fmt.Printf("Error sending game start message: %v\n", err)
		return
	}

	// Store the message ID
	game.SetMessageID(i.ID)
}

// HandleButton handles button interactions for blackjack games
func (m *Manager) HandleButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID
	customID := i.MessageComponentData().CustomID

	// Get game
	m.mu.RLock()
	game, exists := m.games[channelID]
	m.mu.RUnlock()

	if !exists {
		utils.SendErrorResponse(s, i, utils.NewGameError(utils.ErrGameNotFound, "No game found in this channel"))
		return
	}

	switch customID {
	case "blackjack_join":
		game.HandleJoin(s, i)
	case "blackjack_start":
		game.HandleStart(s, i)
	case "blackjack_hit":
		game.HandleHit(s, i)
	case "blackjack_stand":
		game.HandleStand(s, i)
	default:
		fmt.Printf("Unknown button interaction: %s\n", customID)
	}

	// Clean up finished games
	if game.IsFinished() {
		m.mu.Lock()
		delete(m.games, channelID)
		m.mu.Unlock()
	}
}
