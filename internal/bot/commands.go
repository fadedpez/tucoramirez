package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// commands defines all slash commands for the bot
var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "blackjack",
		Description: "Start a game of blackjack",
	},
	{
		Name:        "dueltuco",
		Description: "Challenge Tuco to a duel",
	},
	// Add more commands here
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
	for _, cmd := range commands {
		registeredCmd, err := b.session.ApplicationCommandCreate(b.config.AppID, b.config.GuildID, cmd)
		if err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.Name, err)
		}
		b.commands = append(b.commands, registeredCmd)
		fmt.Printf("Registered command: %s\n", cmd.Name)
	}

	return nil
}

// cleanupCommands removes all registered commands
func (b *Bot) cleanupCommands() error {
	// Get existing commands
	existingCommands, err := b.session.ApplicationCommands(b.config.AppID, b.config.GuildID)
	if err != nil {
		return fmt.Errorf("failed to fetch existing commands: %w", err)
	}

	// Delete each command
	for _, cmd := range existingCommands {
		if err := b.session.ApplicationCommandDelete(b.config.AppID, b.config.GuildID, cmd.ID); err != nil {
			fmt.Printf("Failed to delete command %s: %v\n", cmd.Name, err)
		} else {
			fmt.Printf("Deleted command: %s\n", cmd.Name)
		}
	}

	return nil
}
