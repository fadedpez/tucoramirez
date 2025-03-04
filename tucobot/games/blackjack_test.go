package games

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

// MockSession implements sessionHandler for testing
type MockSession struct {
	interactionResponse *discordgo.InteractionResponse
	messageSent        *discordgo.MessageSend
}

func (m *MockSession) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	m.interactionResponse = r
	return nil
}

func (m *MockSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.messageSent = data
	return &discordgo.Message{}, nil
}

func TestStartBlackjackGame(t *testing.T) {
	mock := &MockSession{}
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID:       "test-user",
					Username: "TestUser",
				},
			},
		},
	}

	StartBlackjackGame(mock, interaction)

	if mock.interactionResponse == nil {
		t.Fatal("Expected interaction response, got nil")
	}

	if mock.interactionResponse.Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Errorf("Expected response type %d, got %d", discordgo.InteractionResponseChannelMessageWithSource, mock.interactionResponse.Type)
	}

	components := mock.interactionResponse.Data.Components
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

	hitButton, ok := row.Components[0].(discordgo.Button)
	if !ok {
		t.Fatal("Expected Button component for Hit")
	}
	if hitButton.Label != "Hit" {
		t.Errorf("Expected Hit button, got: %s", hitButton.Label)
	}

	standButton, ok := row.Components[1].(discordgo.Button)
	if !ok {
		t.Fatal("Expected Button component for Stand")
	}
	if standButton.Label != "Stand" {
		t.Errorf("Expected Stand button, got: %s", standButton.Label)
	}
}

func TestHandleBlackjackButton(t *testing.T) {
	mock := &MockSession{}
	interaction := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID:       "test-user",
					Username: "TestUser",
				},
			},
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "blackjack_hit",
			},
		},
	}

	// Start a new game
	StartBlackjackGame(mock, &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID:       "test-user",
					Username: "TestUser",
				},
			},
		},
	})

	// Test Hit button
	HandleBlackjackButton(mock, interaction)

	if mock.interactionResponse == nil {
		t.Fatal("Expected interaction response, got nil")
	}

	if mock.interactionResponse.Type != discordgo.InteractionResponseUpdateMessage {
		t.Errorf("Expected response type %d, got %d", discordgo.InteractionResponseUpdateMessage, mock.interactionResponse.Type)
	}
}
