package commands

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/assert"
)

// TestStatsCommand_Command tests the Command method of StatsCommand
func TestStatsCommand_Command(t *testing.T) {
	// Create the command
	cmd := &StatsCommand{
		statisticsService: nil, // We don't actually use the service in this test
		blackjackCommand:  nil, // We don't actually use the command in this test
	}

	// Call the method being tested
	command := cmd.Command()

	// Assert that the command was created correctly
	assert.Equal(t, "stats", command.Name)
	assert.Equal(t, "View player statistics", command.Description)
	assert.Equal(t, 1, len(command.Options))
	assert.Equal(t, "blackjack", command.Options[0].Name)
	assert.Equal(t, "View blackjack statistics", command.Options[0].Description)
	assert.Equal(t, discordgo.ApplicationCommandOptionSubCommand, command.Options[0].Type)
}

// TestStatsCommand_Handle tests the Handle method of StatsCommand
func TestStatsCommand_Handle(t *testing.T) {
	// Skip this test for now as it requires more complex mocking
	t.Skip("Skipping test that requires complex mocking")
}

// TestStatsCommand_HandleComponentInteraction tests the HandleComponentInteraction method
func TestStatsCommand_HandleComponentInteraction(t *testing.T) {
	// Skip this test for now as it requires more complex mocking
	t.Skip("Skipping test that requires complex mocking")
}
