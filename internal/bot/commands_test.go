package bot

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type CommandsTestSuite struct {
	suite.Suite
}

func TestCommandsSuite(t *testing.T) {
	suite.Run(t, new(CommandsTestSuite))
}

func (s *CommandsTestSuite) TestCommands() {
	// Test that Commands slice is properly initialized
	s.NotNil(Commands, "Commands slice should not be nil")
	s.NotEmpty(Commands, "Commands slice should not be empty")

	// Test individual commands
	commandNames := make(map[string]bool)
	for _, cmd := range Commands {
		// Verify command properties
		s.NotEmpty(cmd.Name, "Command name should not be empty")
		s.NotEmpty(cmd.Description, "Command description should not be empty")

		// Check for duplicate commands
		s.False(commandNames[cmd.Name], "Command names should be unique")
		commandNames[cmd.Name] = true
	}

	// Verify required commands exist
	requiredCommands := []string{"blackjack", "dueltuco"}
	for _, required := range requiredCommands {
		s.True(commandNames[required], "Required command %s should exist", required)
	}
}
