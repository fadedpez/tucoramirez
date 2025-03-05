package bot

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/config"
	"github.com/fadedpez/tucoramirez/pkg/games/blackjack"
)

// Bot represents the Discord bot and its dependencies
type Bot struct {
	config     *config.Config
	session    *discordgo.Session
	commands   []*discordgo.ApplicationCommand
	blackjack  *blackjack.Manager
	shutdownWg sync.WaitGroup
}

// New creates a new instance of Bot
func New(cfg *config.Config) (*Bot, error) {
	// Create Discord session
	session, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create Discord session: %w", err)
	}

	// Create bot instance
	bot := &Bot{
		config:   cfg,
		session:  session,
		commands: make([]*discordgo.ApplicationCommand, 0),
	}

	// Initialize game managers
	bot.blackjack = blackjack.NewManager(session)

	// Register handlers
	session.AddHandler(bot.handleInteractionCreate)
	session.AddHandler(bot.handleMessageCreate)

	return bot, nil
}

// Start initializes the bot and connects to Discord
func (b *Bot) Start() error {
	// Open connection to Discord
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("failed to open Discord connection: %w", err)
	}

	// Register commands
	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the bot
func (b *Bot) Shutdown() {
	// Cleanup commands if in development
	if b.config.IsDevelopment() {
		b.cleanupCommands()
	}

	// Close Discord session
	if err := b.session.Close(); err != nil {
		fmt.Printf("Error closing Discord session: %v\n", err)
	}

	// Wait for any ongoing operations to complete
	b.shutdownWg.Wait()
}

// handleInteractionCreate handles Discord interaction events
func (b *Bot) handleInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleSlashCommand(s, i)
	case discordgo.InteractionMessageComponent:
		b.handleMessageComponent(s, i)
	}
}

// handleMessageCreate handles Discord message events
func (b *Bot) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Handle message commands
	b.handleMessageCommands(s, m)
}

// Additional handler methods will be defined in handlers.go
