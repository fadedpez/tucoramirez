package bot

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/config"
	discordmock "github.com/fadedpez/tucoramirez/internal/discord/mock"
	"github.com/fadedpez/tucoramirez/internal/games"
	storagemock "github.com/fadedpez/tucoramirez/pkg/storage/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockConfig struct {
	config.Config
	isDev bool
}

func (c *mockConfig) IsDevelopment() bool {
	return c.isDev
}

type BotTestSuite struct {
	suite.Suite
	session *discordmock.SessionHandler
	config  *mockConfig
	storage *storagemock.Storage
	bot     *Bot
}

func TestBotSuite(t *testing.T) {
	suite.Run(t, new(BotTestSuite))
}

func (s *BotTestSuite) SetupTest() {
	s.session = &discordmock.SessionHandler{}
	s.session.Test(s.T())
	s.storage = storagemock.New()
	s.config = &mockConfig{
		Config: config.Config{
			AppID:    "test-app-id",
			GuildID:  "test-guild-id",
			DataDir:  "/tmp/tucobot-test",
		},
		isDev: true, // Set development mode
	}

	// Mock AddHandler calls
	mockHandlerFunc := func() func() { return func() {} }
	s.session.On("AddHandler", mock.AnythingOfType("func(*discordgo.Session, *discordgo.InteractionCreate)")).Return(mockHandlerFunc())
	s.session.On("AddHandler", mock.AnythingOfType("func(*discordgo.Session, *discordgo.MessageCreate)")).Return(mockHandlerFunc())

	// Create bot with mock storage
	s.bot = &Bot{
		config:      &s.config.Config,
		session:     s.session,
		commands:    make([]*discordgo.ApplicationCommand, 0),
		registry:    games.NewRegistry(),
		managers:    make(map[string]games.Manager),
		storage:     s.storage,
		cleanupStop: make(chan struct{}),
	}

	// Register games
	err := s.bot.registerGames()
	s.Require().NoError(err)

	// Register handlers
	s.bot.registerHandlers()
}

func (s *BotTestSuite) TestRegisterCommands() {
	// Setup
	// Mock session Open
	s.session.On("Open").Return(nil).Maybe()

	// Mock existing commands for cleanup (since we're in dev mode)
	existingCmd := &discordgo.ApplicationCommand{
		ID:   "existing-cmd-id",
		Name: "existing-cmd",
	}
	existingCmds := []*discordgo.ApplicationCommand{existingCmd}
	s.session.On("ApplicationCommands", s.config.AppID, s.config.GuildID).
		Return(existingCmds, nil).Maybe()

	s.session.On("ApplicationCommandDelete", s.config.AppID, s.config.GuildID, existingCmd.ID).
		Return(nil).Maybe()

	// Mock command registration
	registeredCmd := &discordgo.ApplicationCommand{
		ID:          "new-cmd-id",
		Name:        "test-cmd",
		Description: "test description",
	}
	s.session.On("ApplicationCommandCreate", s.config.AppID, s.config.GuildID, mock.Anything).
		Return(registeredCmd, nil).Maybe()

	// Execute
	err := s.bot.Start()

	// Assert
	s.Require().NoError(err)
	s.session.AssertExpectations(s.T())
	s.Equal(len(Commands), len(s.bot.commands))
}

func (s *BotTestSuite) TestRegisterCommandsError() {
	// Setup
	// Mock session Open
	s.session.On("Open").Return(nil).Maybe()

	// Mock existing commands for cleanup (since we're in dev mode)
	emptyCommands := make([]*discordgo.ApplicationCommand, 0)
	s.session.On("ApplicationCommands", s.config.AppID, s.config.GuildID).
		Return(emptyCommands, nil).Maybe()

	// Mock command registration error for the first command
	s.session.On("ApplicationCommandCreate", s.config.AppID, s.config.GuildID, mock.Anything).
		Return(&discordgo.ApplicationCommand{}, assert.AnError).Maybe()

	// Execute
	err := s.bot.Start()

	// Assert
	s.Require().Error(err)
	s.session.AssertExpectations(s.T())
	s.Empty(s.bot.commands)
}

func (s *BotTestSuite) TestCleanupCommands() {
	// Setup
	existingCmds := []*discordgo.ApplicationCommand{
		{ID: "cmd1", Name: "test1"},
		{ID: "cmd2", Name: "test2"},
	}
	s.session.On("ApplicationCommands", s.config.AppID, s.config.GuildID).
		Return(existingCmds, nil).Maybe()

	for _, cmd := range existingCmds {
		s.session.On("ApplicationCommandDelete", s.config.AppID, s.config.GuildID, cmd.ID).
			Return(nil).Maybe()
	}

	// Execute
	err := s.bot.cleanupCommands()

	// Assert
	s.Require().NoError(err)
	s.session.AssertExpectations(s.T())
}

func (s *BotTestSuite) TestCleanupCommandsError() {
	// Setup
	emptyCommands := make([]*discordgo.ApplicationCommand, 0)
	s.session.On("ApplicationCommands", s.config.AppID, s.config.GuildID).
		Return(emptyCommands, assert.AnError).Maybe()

	// Execute
	err := s.bot.cleanupCommands()

	// Assert
	s.Require().Error(err)
	s.session.AssertExpectations(s.T())
}
