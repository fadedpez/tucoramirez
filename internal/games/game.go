package games

import (
	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/discord"
)

// Game represents a game that can be played through Discord
type Game interface {
	// HandleStart handles the start of a new game
	HandleStart(s discord.SessionHandler, i *discordgo.InteractionCreate)

	// HandleButton handles button interactions for the game
	HandleButton(s discord.SessionHandler, i *discordgo.InteractionCreate)

	// IsFinished returns whether the game is finished
	IsFinished() bool

	// String returns a string representation of the game state
	String() string

	// GetButtons returns the appropriate button components for the current game state
	GetButtons() []discordgo.MessageComponent
}

// Manager represents a game manager that handles game instances
type Manager interface {
	// HandleStart handles the start of a new game
	HandleStart(s discord.SessionHandler, i *discordgo.InteractionCreate)

	// HandleButton handles button interactions for games
	HandleButton(s discord.SessionHandler, i *discordgo.InteractionCreate)
}

// Factory creates new game instances
type Factory interface {
	// CreateGame creates a new game instance
	CreateGame(creatorID, channelID string, players []string) Game

	// CreateManager creates a new game manager
	CreateManager() Manager
}
