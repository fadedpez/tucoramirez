package discord

import (
	"github.com/bwmarrin/discordgo"
)

// SessionHandler defines the interface for Discord session operations
type SessionHandler interface {
	// Core interaction methods
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse) error
	ChannelMessageSend(channelID string, content string) (*discordgo.Message, error)
	ChannelMessageEdit(channelID string, messageID string, content string) (*discordgo.Message, error)
	ChannelMessageEditComplex(m *discordgo.MessageEdit) (*discordgo.Message, error)
	
	// Application command methods
	ApplicationCommandCreate(appID string, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error)
	ApplicationCommandDelete(appID string, guildID string, cmdID string) error
	ApplicationCommands(appID string, guildID string) ([]*discordgo.ApplicationCommand, error)

	// Session methods
	Open() error
	Close() error
	AddHandler(handler interface{}) func()

	// State methods
	State() *discordgo.State
}

// DiscordSession implements SessionHandler using discordgo.Session
type DiscordSession struct {
	*discordgo.Session
}

// NewSession creates a new DiscordSession
func NewSession(token string) (*DiscordSession, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}
	return &DiscordSession{Session: s}, nil
}

// Ensure DiscordSession implements SessionHandler
var _ SessionHandler = (*DiscordSession)(nil)

// InteractionRespond implements SessionHandler
func (s *DiscordSession) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse) error {
	return s.Session.InteractionRespond(i, r)
}

// ChannelMessageSend implements SessionHandler
func (s *DiscordSession) ChannelMessageSend(channelID string, content string) (*discordgo.Message, error) {
	return s.Session.ChannelMessageSend(channelID, content)
}

// ChannelMessageEdit implements SessionHandler
func (s *DiscordSession) ChannelMessageEdit(channelID string, messageID string, content string) (*discordgo.Message, error) {
	return s.Session.ChannelMessageEdit(channelID, messageID, content)
}

// ChannelMessageEditComplex implements SessionHandler
func (s *DiscordSession) ChannelMessageEditComplex(m *discordgo.MessageEdit) (*discordgo.Message, error) {
	return s.Session.ChannelMessageEditComplex(m)
}

// ApplicationCommandCreate implements SessionHandler
func (s *DiscordSession) ApplicationCommandCreate(appID string, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	return s.Session.ApplicationCommandCreate(appID, guildID, cmd, options...)
}

// ApplicationCommandDelete implements SessionHandler
func (s *DiscordSession) ApplicationCommandDelete(appID string, guildID string, cmdID string) error {
	return s.Session.ApplicationCommandDelete(appID, guildID, cmdID)
}

// ApplicationCommands implements SessionHandler
func (s *DiscordSession) ApplicationCommands(appID string, guildID string) ([]*discordgo.ApplicationCommand, error) {
	return s.Session.ApplicationCommands(appID, guildID)
}

// Open implements SessionHandler
func (s *DiscordSession) Open() error {
	return s.Session.Open()
}

// Close implements SessionHandler
func (s *DiscordSession) Close() error {
	return s.Session.Close()
}

// AddHandler implements SessionHandler
func (s *DiscordSession) AddHandler(handler interface{}) func() {
	return s.Session.AddHandler(handler)
}

// State implements SessionHandler
func (s *DiscordSession) State() *discordgo.State {
	return s.Session.State
}
