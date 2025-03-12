package discord

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/pkg/repositories/game"
	"github.com/fadedpez/tucoramirez/pkg/services/blackjack"
)

type GameLobby struct {
	OwnerID string
	Players map[string]bool // playerID -> joined
}

// Bot represents the Discord bot instance
type Bot struct {
	session *discordgo.Session
	token   string

	// Protected maps for game state
	mu      sync.RWMutex // Protects games and lobbies maps
	games   map[string]*blackjack.Game
	lobbies map[string]*GameLobby

	// Storage repository
	repo game.Repository
}

// NewBot creates a new instance of the bot
func NewBot(token string) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	bot := &Bot{
		session: session,
		token:   token,
		games:   make(map[string]*blackjack.Game),
		lobbies: make(map[string]*GameLobby),
		repo:    game.NewMemoryRepository(),  // Initialize the repository
	}

	// Register handlers
	session.AddHandler(bot.handleReady)
	session.AddHandler(bot.handleInteractions)

	// Identify the intents we need
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions

	return bot, nil
}

// Start initializes the bot and connects to Discord
func (b *Bot) Start() error {
	// Clean up any stale state
	b.mu.Lock()
	b.games = make(map[string]*blackjack.Game)
	b.lobbies = make(map[string]*GameLobby)
	b.mu.Unlock()

	// Open websocket connection
	err := b.session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}

	// Register slash commands
	_, err = b.session.ApplicationCommandCreate(b.session.State.User.ID, "", &discordgo.ApplicationCommand{
		Name:        "blackjack",
		Description: "Start a new game of blackjack!",
	})
	if err != nil {
		return fmt.Errorf("error creating command: %w", err)
	}

	return nil
}

// Stop gracefully shuts down the bot and closes the Discord connection
func (b *Bot) Stop() error {
	// Clean up any active games and lobbies
	b.mu.Lock()
	b.games = make(map[string]*blackjack.Game)
	b.lobbies = make(map[string]*GameLobby)
	b.mu.Unlock()

	// Close websocket connection
	if err := b.session.Close(); err != nil {
		return fmt.Errorf("error closing connection: %w", err)
	}

	return nil
}
