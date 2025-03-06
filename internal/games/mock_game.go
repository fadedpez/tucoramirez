package games

import (
	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/discord"
	"github.com/stretchr/testify/mock"
)

// MockGame implements Game for testing
type MockGame struct {
	mock.Mock
}

func (m *MockGame) HandleStart(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	m.Called(s, i)
}

func (m *MockGame) HandleButton(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	m.Called(s, i)
}

func (m *MockGame) IsFinished() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockGame) String() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockGame) GetButtons() []discordgo.MessageComponent {
	args := m.Called()
	return args.Get(0).([]discordgo.MessageComponent)
}

// MockManager implements Manager for testing
type MockManager struct {
	mock.Mock
}

func (m *MockManager) HandleStart(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	m.Called(s, i)
}

func (m *MockManager) HandleButton(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	m.Called(s, i)
}

// MockFactory implements Factory for testing
type MockFactory struct {
	mock.Mock
}

func (m *MockFactory) CreateGame(creatorID, channelID string, players []string) Game {
	args := m.Called(creatorID, channelID, players)
	return args.Get(0).(Game)
}

func (m *MockFactory) CreateManager() Manager {
	args := m.Called()
	return args.Get(0).(Manager)
}
