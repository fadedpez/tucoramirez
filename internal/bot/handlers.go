package bot

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleSlashCommand handles all slash commands
func (b *Bot) handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Name {
	case "blackjack":
		b.blackjack.HandleStart(s, i)
	case "dueltuco":
		b.handleDuelTuco(s, i)
	default:
		fmt.Printf("Unknown command: %s\n", i.ApplicationCommandData().Name)
	}
}

// handleMessageComponent handles button clicks and other message components
func (b *Bot) handleMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	switch {
	case strings.HasPrefix(customID, "blackjack_"):
		b.blackjack.HandleButton(s, i)
	case strings.HasPrefix(customID, "duel_"):
		b.handleDuelButton(s, i)
	default:
		fmt.Printf("Unknown component interaction: %s\n", customID)
	}
}

// handleMessageCommands handles text-based commands
func (b *Bot) handleMessageCommands(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Handle "tuco?" command
	if regexp.MustCompile(`(?i)tuco\?$`).MatchString(m.Content) {
		b.handleTucoImage(s, m)
		return
	}

	// Add more message-based commands here
}

// handleDuelTuco handles the duel command
func (b *Bot) handleDuelTuco(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Implementation will be moved here from tuco.go
}

// handleDuelButton handles duel-related button interactions
func (b *Bot) handleDuelButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Implementation will be moved here from tuco.go
}

// handleTucoImage handles the "tuco?" command
func (b *Bot) handleTucoImage(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Implementation will be moved here from tuco.go
}
