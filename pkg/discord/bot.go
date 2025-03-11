package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/pkg/services/blackjack"
)

type Bot struct {
	session *discordgo.Session
	games   map[string]*blackjack.Game // channelID -> Game
}

func New(token string) (*Bot, error) {
	// Initialize Discord session
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	// Create new bot instance

	bot := &Bot{
		session: session,
		games:   make(map[string]*blackjack.Game),
	}

	// Setup command handlers
	session.AddHandler(bot.handleReady)
	session.AddHandler(bot.handleInteractions)

	// Identify the intents we need

	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions
	// Return bot instance
	return bot, nil
}

// Add to bot.go after the New function:

// Start opens a websocket connection and begins listening for Discord events
func (b *Bot) Start() error {
	// Open websocket connection to Discord
	err := b.session.Open()
	if err != nil {
		return fmt.Errorf("error opening connection: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the bot and closes the Discord connection
func (b *Bot) Stop() error {
	// Clean up any active games
	b.games = make(map[string]*blackjack.Game)

	// Close websocket connection
	err := b.session.Close()
	if err != nil {
		return fmt.Errorf("error closing connection: %w", err)
	}
	return nil
}
