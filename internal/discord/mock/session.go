package mock

import (
	"github.com/bwmarrin/discordgo"
	"github.com/stretchr/testify/mock"
)

// SessionHandler is a mock implementation of discord.SessionHandler
type SessionHandler struct {
	mock.Mock
}

// InteractionRespond implements discord.SessionHandler
func (s *SessionHandler) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse) error {
	args := s.Called(i, r)
	return args.Error(0)
}

// ChannelMessageSend implements discord.SessionHandler
func (s *SessionHandler) ChannelMessageSend(channelID string, content string) (*discordgo.Message, error) {
	args := s.Called(channelID, content)
	return args.Get(0).(*discordgo.Message), args.Error(1)
}

// ChannelMessageEdit implements discord.SessionHandler
func (s *SessionHandler) ChannelMessageEdit(channelID string, messageID string, content string) (*discordgo.Message, error) {
	args := s.Called(channelID, messageID, content)
	return args.Get(0).(*discordgo.Message), args.Error(1)
}

// ChannelMessageEditComplex implements discord.SessionHandler
func (s *SessionHandler) ChannelMessageEditComplex(m *discordgo.MessageEdit) (*discordgo.Message, error) {
	args := s.Called(m)
	return args.Get(0).(*discordgo.Message), args.Error(1)
}

// ApplicationCommandCreate implements discord.SessionHandler
func (s *SessionHandler) ApplicationCommandCreate(appID string, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	args := s.Called(appID, guildID, cmd)
	return args.Get(0).(*discordgo.ApplicationCommand), args.Error(1)
}

// ApplicationCommandDelete implements discord.SessionHandler
func (s *SessionHandler) ApplicationCommandDelete(appID string, guildID string, cmdID string) error {
	args := s.Called(appID, guildID, cmdID)
	return args.Error(0)
}

// ApplicationCommands implements discord.SessionHandler
func (s *SessionHandler) ApplicationCommands(appID string, guildID string) ([]*discordgo.ApplicationCommand, error) {
	args := s.Called(appID, guildID)
	return args.Get(0).([]*discordgo.ApplicationCommand), args.Error(1)
}

// Open implements discord.SessionHandler
func (s *SessionHandler) Open() error {
	args := s.Called()
	return args.Error(0)
}

// Close implements discord.SessionHandler
func (s *SessionHandler) Close() error {
	args := s.Called()
	return args.Error(0)
}

// AddHandler implements discord.SessionHandler
func (s *SessionHandler) AddHandler(handler interface{}) func() {
	args := s.Called(handler)
	return args.Get(0).(func())
}

// State implements discord.SessionHandler
func (s *SessionHandler) State() *discordgo.State {
	args := s.Called()
	return args.Get(0).(*discordgo.State)
}
