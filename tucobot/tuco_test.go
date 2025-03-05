package tucobot

import (
	"testing"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// MockSession implements sessionHandler for testing
type MockSession struct {
	commands            []*discordgo.ApplicationCommand
	interactionResponse *discordgo.InteractionResponse
	messageSent         *discordgo.MessageSend
	followupMessage     *discordgo.WebhookParams
	editedMessage       *discordgo.WebhookEdit
	editedChannelMessage string
}

func (m *MockSession) ApplicationCommandCreate(appID, guildID string, cmd *discordgo.ApplicationCommand, options ...discordgo.RequestOption) (*discordgo.ApplicationCommand, error) {
	m.commands = append(m.commands, cmd)
	return cmd, nil
}

func (m *MockSession) ApplicationCommands(appID, guildID string, options ...discordgo.RequestOption) ([]*discordgo.ApplicationCommand, error) {
	return m.commands, nil
}

func (m *MockSession) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	m.interactionResponse = r
	return nil
}

func (m *MockSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.messageSent = data
	return &discordgo.Message{ID: "test_message_id"}, nil
}

func (m *MockSession) ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error {
	return nil
}

func (m *MockSession) FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.followupMessage = data
	return &discordgo.Message{}, nil
}

func (m *MockSession) InteractionResponseEdit(i *discordgo.Interaction, edit *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.editedMessage = edit
	return &discordgo.Message{}, nil
}

func (m *MockSession) ChannelMessageEdit(channelID string, messageID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.editedChannelMessage = content
	return &discordgo.Message{}, nil
}

func TestRegisterCommands(t *testing.T) {
	mock := &MockSession{}
	RegisterCommands(mock, "test-app-id", "test-guild-id")

	if len(mock.commands) != 2 {
		t.Errorf("Expected 2 commands to be registered, got %d", len(mock.commands))
	}

	expectedCommands := map[string]bool{
		"dueltuco":  false,
		"blackjack": false,
	}

	for _, cmd := range mock.commands {
		if _, ok := expectedCommands[cmd.Name]; !ok {
			t.Errorf("Unexpected command registered: %s", cmd.Name)
		}
		expectedCommands[cmd.Name] = true
	}

	for name, found := range expectedCommands {
		if !found {
			t.Errorf("Expected command %s was not registered", name)
		}
	}
}

func TestInteractionCreate_Command(t *testing.T) {
	mock := &MockSession{}
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "dueltuco",
			},
		},
	}

	InteractionCreate(mock, interaction)

	if mock.interactionResponse == nil {
		t.Fatal("Expected interaction response, got nil")
	}

	if mock.interactionResponse.Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Errorf("Expected response type %d, got %d", discordgo.InteractionResponseChannelMessageWithSource, mock.interactionResponse.Type)
	}
}

func TestInteractionCreate_ButtonClick(t *testing.T) {
	mock := &MockSession{}
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "duel_accept",
			},
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID:       "test-user",
					Username: "Test User",
				},
			},
		},
	}

	InteractionCreate(mock, interaction)

	if mock.interactionResponse == nil {
		t.Fatal("Expected interaction response, got nil")
	}

	if mock.interactionResponse.Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Errorf("Expected response type %d, got %d", discordgo.InteractionResponseChannelMessageWithSource, mock.interactionResponse.Type)
	}

	if !strings.Contains(mock.interactionResponse.Data.Content, "Test User") {
		t.Error("Expected response to contain username")
	}
}
