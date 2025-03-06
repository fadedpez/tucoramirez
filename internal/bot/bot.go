package bot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/config"
	"github.com/fadedpez/tucoramirez/internal/discord"
	"github.com/fadedpez/tucoramirez/internal/games"
	"github.com/fadedpez/tucoramirez/internal/logging"
	"github.com/fadedpez/tucoramirez/pkg/games/blackjack"
	"github.com/fadedpez/tucoramirez/pkg/storage"
	"github.com/fadedpez/tucoramirez/pkg/storage/file"
)

// Bot represents the Discord bot
type Bot struct {
	config      *config.Config
	session     discord.SessionHandler
	commands    []*discordgo.ApplicationCommand
	registry    *games.Registry
	managers    map[string]games.Manager
	storage     storage.Storage
	shutdownWg  sync.WaitGroup
	cleanupStop chan struct{}
}

// NewWithSession creates a new bot instance with the provided session
func NewWithSession(cfg *config.Config, session discord.SessionHandler) (*Bot, error) {
	// Create storage
	storageOptions := storage.NewOptions()
	storageOptions.Path = filepath.Join(cfg.DataDir, "games")
	storageOptions.MaxGameAge = 24 * time.Hour
	storageOptions.AutoCleanup = true

	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storageOptions.Path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	fileStorage, err := file.New(storageOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	// Create bot instance
	bot := &Bot{
		config:      cfg,
		session:     session,
		commands:    make([]*discordgo.ApplicationCommand, 0),
		registry:    games.NewRegistry(),
		managers:    make(map[string]games.Manager),
		storage:     fileStorage,
		cleanupStop: make(chan struct{}),
	}

	// Register games
	if err := bot.registerGames(); err != nil {
		return nil, fmt.Errorf("failed to register games: %w", err)
	}

	// Register command handlers
	bot.registerHandlers()

	return bot, nil
}

// New creates a new instance of Bot
func New(cfg *config.Config) (*Bot, error) {
	// Create Discord session
	session, err := discord.NewSession(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	return NewWithSession(cfg, session)
}

// registerGames registers all available games with the registry
func (b *Bot) registerGames() error {
	// Register blackjack
	blackjackFactory := blackjack.NewFactory(b.session, b.storage)
	if err := b.registry.RegisterGame("blackjack", blackjackFactory); err != nil {
		return fmt.Errorf("failed to register blackjack: %w", err)
	}

	return nil
}

// registerHandlers registers all command handlers
func (b *Bot) registerHandlers() {
	// Add interaction handler
	b.session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			b.handleSlashCommand(b.session, i)
		case discordgo.InteractionMessageComponent:
			b.handleButton(b.session, i)
		}
	})

	// Add message handler
	b.session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == b.session.State().User.ID {
			return
		}
		b.handleMessage(b.session, m)
	})
}

// Start starts the bot
func (b *Bot) Start() error {
	// Open Discord session
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open session: %w", err)
	}

	// Register commands
	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	// Start cleanup goroutine
	b.startCleanup()

	return nil
}

// Stop stops the bot and cleans up resources
func (b *Bot) Stop() error {
	// Stop cleanup goroutine
	close(b.cleanupStop)

	// Remove commands
	if err := b.cleanupCommands(); err != nil {
		logging.Default.Error("Failed to cleanup commands: %v", err)
	}

	// Close session
	if err := b.session.Close(); err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	// Wait for all operations to complete
	b.shutdownWg.Wait()

	return nil
}

// Shutdown gracefully shuts down the bot
func (b *Bot) Shutdown() {
	b.Stop()
}

// handleInteractionCreate handles Discord interaction events
func (b *Bot) handleInteractionCreate(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	// Increment wait group
	b.shutdownWg.Add(1)
	defer b.shutdownWg.Done()

	// Handle the interaction
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleSlashCommand(b.session, i)
	case discordgo.InteractionMessageComponent:
		b.handleButton(b.session, i)
	}
}

// handleMessageCreate handles Discord message events
func (b *Bot) handleMessageCreate(s discord.SessionHandler, m *discordgo.MessageCreate) {
	// Increment wait group
	b.shutdownWg.Add(1)
	defer b.shutdownWg.Done()

	// Ignore messages from the bot itself
	if m.Author.ID == s.State().User.ID {
		return
	}

	// Handle message commands
	b.handleMessageCommands(s, m)
}

// startCleanup starts a goroutine to periodically clean up finished games
func (b *Bot) startCleanup() {
	b.shutdownWg.Add(1)
	go func() {
		defer b.shutdownWg.Done()

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := b.storage.CleanupOldGames(context.Background(), 24*time.Hour); err != nil {
					logging.Default.Error("Failed to cleanup old games: %v", err)
				}
			case <-b.cleanupStop:
				return
			}
		}
	}()
}

// registerCommands registers all slash commands with Discord
func (b *Bot) registerCommands() error {
	// First, clean up existing commands in development
	if b.config.IsDevelopment() {
		if err := b.cleanupCommands(); err != nil {
			return fmt.Errorf("failed to cleanup commands: %w", err)
		}
	}

	// Register each command
	for _, cmd := range Commands {
		registeredCmd, err := b.session.ApplicationCommandCreate(b.config.AppID, b.config.GuildID, cmd)
		if err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.Name, err)
		}
		b.commands = append(b.commands, registeredCmd)
		fmt.Printf("Registered command: %s\n", cmd.Name)
	}

	return nil
}

// cleanupCommands removes all registered slash commands
func (b *Bot) cleanupCommands() error {
	// Get existing commands
	existingCommands, err := b.session.ApplicationCommands(b.config.AppID, b.config.GuildID)
	if err != nil {
		return fmt.Errorf("failed to fetch existing commands: %w", err)
	}

	// Remove each command
	for _, cmd := range existingCommands {
		if err := b.session.ApplicationCommandDelete(b.config.AppID, b.config.GuildID, cmd.ID); err != nil {
			return fmt.Errorf("failed to delete command %s: %w", cmd.Name, err)
		}
		fmt.Printf("Deleted command: %s\n", cmd.Name)
	}

	return nil
}
