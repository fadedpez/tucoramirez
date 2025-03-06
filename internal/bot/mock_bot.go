package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/discord"
	"github.com/fadedpez/tucoramirez/internal/games"
	"github.com/stretchr/testify/mock"
)

// MockBot implements Bot for testing
type MockBot struct {
	mock.Mock
	managers map[string]games.Manager
}

func NewMockBot() *MockBot {
	return &MockBot{
		managers: make(map[string]games.Manager),
	}
}

func (m *MockBot) handleSlashCommand(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	m.Called(s, i)
}

func (m *MockBot) handleButton(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	m.Called(s, i)
}

func (m *MockBot) handleMessage(s discord.SessionHandler, message *discordgo.MessageCreate) {
	m.Called(s, message)
}

func (m *MockBot) handleDuelTuco(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	m.Called(s, i)
}

func (m *MockBot) handleDuelButton(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	m.Called(s, i)
}

func (m *MockBot) handleTucoImage(s discord.SessionHandler, msg *discordgo.MessageCreate) {
	m.Called(s, msg)
}

func (m *MockBot) handleMessageCommands(s discord.SessionHandler, msg *discordgo.MessageCreate) {
	m.Called(s, msg)
}
