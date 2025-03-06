package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/discord"
	"github.com/fadedpez/tucoramirez/internal/types"
)

// handleSlashCommand handles all slash commands
func (b *Bot) handleSlashCommand(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	cmd := i.ApplicationCommandData().Name

	// Check if it's a game command
	if manager, ok := b.managers[cmd]; ok {
		manager.HandleStart(s, i)
		return
	}

	// Handle non-game commands
	switch cmd {
	case "dueltuco":
		b.handleDuelTuco(s, i)
	default:
		err := types.NewGameError(types.ErrInvalidAction, fmt.Sprintf("Unknown command: %s", cmd))
		discord.SendErrorResponse(s, i, err)
	}
}

// handleButton handles button clicks and other message components
func (b *Bot) handleButton(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	// Extract game name from button ID (format: "gamename_action")
	parts := strings.SplitN(customID, "_", 2)
	if len(parts) != 2 {
		err := types.NewGameError(types.ErrInvalidAction, fmt.Sprintf("Invalid button ID format: %s", customID))
		discord.SendErrorResponse(s, i, err)
		return
	}

	gameName := parts[0]
	switch gameName {
	case "blackjack":
		if manager, ok := b.managers["blackjack"]; ok {
			manager.HandleButton(s, i)
			return
		}
	case "duel":
		b.handleDuelButton(s, i)
		return
	}

	err := types.NewGameError(types.ErrInvalidAction, fmt.Sprintf("Unknown button interaction: %s", customID))
	discord.SendErrorResponse(s, i, err)
}

// handleDuelTuco handles the duel command
func (b *Bot) handleDuelTuco(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	// TODO: Implement duel command
	response := discord.NewResponse("Not implemented yet", nil)
	discord.SendResponse(s, i, response)
}

// handleDuelButton handles duel-related button interactions
func (b *Bot) handleDuelButton(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	// TODO: Implement duel button handling
	response := discord.NewResponse("Not implemented yet", nil)
	discord.SendResponse(s, i, response)
}

// handleMessage handles all message events
func (b *Bot) handleMessage(s discord.SessionHandler, m *discordgo.MessageCreate) {
	// Handle different message commands
	switch m.Content {
	case "tuco?":
		b.handleTucoImage(s, m)
	default:
		// Handle other message commands
		b.handleMessageCommands(s, m)
	}
}

// handleTucoImage handles the "tuco?" command
func (b *Bot) handleTucoImage(s discord.SessionHandler, m *discordgo.MessageCreate) {
	// TODO: Implement image response
	s.ChannelMessageSend(m.ChannelID, "¿Qué pasa?")
}

// handleMessageCommands handles text-based commands
func (b *Bot) handleMessageCommands(s discord.SessionHandler, m *discordgo.MessageCreate) {
	// Handle different message commands
	switch {
	case strings.HasPrefix(m.Content, "!help"):
		s.ChannelMessageSend(m.ChannelID, "Available commands:\n- tuco?\n- !help")
	}
}
