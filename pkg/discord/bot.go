package discord

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/pkg/discord/commands"
	"github.com/fadedpez/tucoramirez/pkg/repositories/game"
	"github.com/fadedpez/tucoramirez/pkg/services/blackjack"
	"github.com/fadedpez/tucoramirez/pkg/services/image"
	"github.com/fadedpez/tucoramirez/pkg/services/statistics"
	"github.com/fadedpez/tucoramirez/pkg/services/wallet"
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

	// Image service for game completion images
	imageService *image.Service

	// Interaction tracking to prevent duplicates
	interactionMu         sync.RWMutex
	processedInteractions map[string]time.Time
	lastCleanupTime       time.Time

	// Wallet service
	walletService *wallet.Service
	
	// Statistics service
	statisticsService *statistics.Service

	// Command handlers
	blackjackCommand interface{}
	walletCommand    interface{}
	statsCommand     *commands.StatsCommand

	// Channel to signal when the bot is ready
	readyChan chan struct{}
}

// blackjackCommandImpl implements the commands.BlackjackCommand interface
type blackjackCommandImpl struct {
	bot *Bot
}

// StartGame implements the commands.BlackjackCommand interface
func (b *blackjackCommandImpl) StartGame(ctx context.Context, s *discordgo.Session, channelID, userID string) error {
	// Create a new game
	game := blackjack.NewGame(userID, b.bot.repo)

	// Add the game to the bot's games map
	b.bot.mu.Lock()
	b.bot.games[channelID] = game
	b.bot.mu.Unlock()

	// Create a new message for the game
	embed := &discordgo.MessageEmbed{
		Title:       "Blackjack Game",
		Description: "A new game of blackjack has started! Click 'Join' to participate.",
		Color:       0x00ff00,
	}

	// Create the join button
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Join",
					Style:    discordgo.PrimaryButton,
					CustomID: "join",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🃏",
					},
				},
			},
		},
	}

	// Send the message
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		return fmt.Errorf("error sending game message: %w", err)
	}

	// Create a lobby for the game
	b.bot.mu.Lock()
	b.bot.lobbies[channelID] = &GameLobby{
		OwnerID: userID,
		Players: make(map[string]bool),
	}
	b.bot.mu.Unlock()

	// Add the owner to the lobby
	b.bot.mu.Lock()
	b.bot.lobbies[channelID].Players[userID] = true
	b.bot.mu.Unlock()

	return nil
}

// NewBot creates a new instance of the bot
func NewBot(token string, repository game.Repository, walletService *wallet.Service, statisticsService *statistics.Service) (*Bot, error) {
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	// Initialize image service
	imageService, err := image.NewService("images.txt")
	if err != nil {
		return nil, fmt.Errorf("error initializing image service: %w", err)
	}

	// Initialize the bot
	bot := &Bot{
		session:               session,
		token:                 token,
		games:                 make(map[string]*blackjack.Game),
		lobbies:               make(map[string]*GameLobby),
		repo:                  repository,
		imageService:          imageService,
		processedInteractions: make(map[string]time.Time),
		lastCleanupTime:       time.Now(),
		walletService:         walletService,
		statisticsService:     statisticsService,
		readyChan:             make(chan struct{}),
	}

	// Initialize the stats command if statistics service is available
	if statisticsService != nil {
		// Create a BlackjackCommand interface implementation
		blackjackCmd := &blackjackCommandImpl{bot: bot}
		// Initialize the stats command
		bot.statsCommand = commands.NewStatsCommand(statisticsService, blackjackCmd)
		bot.blackjackCommand = blackjackCmd
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
	log.Printf("Cleaning up stale state: %d games, %d lobbies", len(b.games), len(b.lobbies))
	b.games = make(map[string]*blackjack.Game)
	b.lobbies = make(map[string]*GameLobby)
	b.mu.Unlock()

	// Log status of command handlers
	if b.statsCommand != nil {
		log.Printf("Stats command is available and will be registered")
	} else {
		log.Printf("Stats command is not available - check statistics service initialization")
	}

	// Open websocket connection
	err := b.session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
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

	// Close repository
	if err := b.repo.Close(); err != nil {
		return fmt.Errorf("error closing repository: %w", err)
	}

	// Close websocket connection
	if err := b.session.Close(); err != nil {
		return fmt.Errorf("error closing connection: %w", err)
	}

	return nil
}
