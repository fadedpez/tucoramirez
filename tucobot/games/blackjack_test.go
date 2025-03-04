package games

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

// MockSession implements sessionHandler for testing
type MockSession struct {
	interactionResponse *discordgo.InteractionResponse
	messageSent         *discordgo.MessageSend
}

func (m *MockSession) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	m.interactionResponse = r
	return nil
}

func (m *MockSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.messageSent = data
	return &discordgo.Message{}, nil
}

func (m *MockSession) ApplicationCommandDelete(appID, guildID, cmdID string, options ...discordgo.RequestOption) error {
	return nil
}

func TestStartBlackjackGame(t *testing.T) {
	mock := &MockSession{}
	channelID := "test-channel"

	err := StartBlackjackGame(mock, channelID)
	if err != nil {
		t.Fatalf("StartBlackjackGame failed: %v", err)
	}

	if mock.messageSent == nil {
		t.Fatal("Expected message to be sent, got nil")
	}

	if mock.messageSent.Content != "A new game of blackjack has started! Click Join to play." {
		t.Errorf("Expected game start message, got: %s", mock.messageSent.Content)
	}

	components := mock.messageSent.Components
	if len(components) != 1 {
		t.Fatalf("Expected 1 component row, got %d", len(components))
	}

	row, ok := components[0].(discordgo.ActionsRow)
	if !ok {
		t.Fatal("Expected ActionsRow component")
	}

	if len(row.Components) != 2 {
		t.Fatalf("Expected 2 buttons, got %d", len(row.Components))
	}

	joinButton, ok := row.Components[0].(discordgo.Button)
	if !ok {
		t.Fatal("Expected Button component for Join")
	}
	if joinButton.Label != "Join" {
		t.Errorf("Expected Join button, got: %s", joinButton.Label)
	}

	dealButton, ok := row.Components[1].(discordgo.Button)
	if !ok {
		t.Fatal("Expected Button component for Deal")
	}
	if dealButton.Label != "Deal" {
		t.Errorf("Expected Deal button, got: %s", dealButton.Label)
	}
}

func TestHandleButton(t *testing.T) {
	mock := &MockSession{}
	channelID := "test-channel"

	// Start a new game
	err := StartBlackjackGame(mock, channelID)
	if err != nil {
		t.Fatalf("StartBlackjackGame failed: %v", err)
	}

	// Test Join button
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type:      discordgo.InteractionMessageComponent,
			ChannelID: channelID,
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID:       "test-user",
					Username: "TestUser",
				},
			},
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "join_button",
			},
		},
	}

	err = HandleButton(mock, interaction)
	if err != nil {
		t.Fatalf("HandleButton failed: %v", err)
	}

	if mock.interactionResponse == nil {
		t.Fatal("Expected interaction response, got nil")
	}

	if mock.interactionResponse.Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Errorf("Expected response type %d, got %d", discordgo.InteractionResponseChannelMessageWithSource, mock.interactionResponse.Type)
	}
}
