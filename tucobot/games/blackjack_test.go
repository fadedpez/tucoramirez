package games

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot/cards"
)

// MockSession implements necessary discordgo.Session methods for testing
type MockSession struct {
	interactionResponse  *discordgo.InteractionResponse
	followupMessage      *discordgo.WebhookParams
	channelMessage       *discordgo.MessageSend
	editedMessage        *discordgo.WebhookEdit
	editedChannelMessage string
}

// Required Session interface methods
func (m *MockSession) InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error {
	m.interactionResponse = r
	return nil
}

func (m *MockSession) FollowupMessageCreate(i *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.followupMessage = data
	return &discordgo.Message{}, nil
}

func (m *MockSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.channelMessage = data
	return &discordgo.Message{ID: "test_message_id"}, nil
}

func (m *MockSession) InteractionResponseEdit(i *discordgo.Interaction, edit *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.editedMessage = edit
	return &discordgo.Message{}, nil
}

func (m *MockSession) ChannelMessageEdit(channelID string, messageID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.editedChannelMessage = content
	return &discordgo.Message{}, nil
}

// Helper function to create a test interaction
func createTestInteraction(userID, username, channelID, customID string) *discordgo.InteractionCreate {
	return &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			ID:        "test-interaction",
			ChannelID: channelID,
			Type:      discordgo.InteractionMessageComponent,
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID:       userID,
					Username: username,
				},
			},
			Data: discordgo.MessageComponentInteractionData{
				CustomID: customID,
			},
		},
	}
}

func TestStartBlackjackGame(t *testing.T) {
	mock := &MockSession{}
	channelID := "test-channel"
	interaction := createTestInteraction("user1", "Player1", channelID, "")

	// Clear any existing games
	activeGames = make(map[string]*BlackjackGame)

	StartBlackjackGame(mock, interaction)

	if mock.interactionResponse == nil {
		t.Fatal("Expected interaction response, got nil")
	}

	// Verify game was created
	game, exists := activeGames[channelID]
	if !exists {
		t.Fatal("Expected game to be created")
	}

	// Check initial game state
	if game.GameState != Waiting {
		t.Error("Game should be in Waiting state")
	}

	// Verify start button
	components := mock.interactionResponse.Data.Components
	if len(components) != 1 {
		t.Fatalf("Expected 1 component row, got %d", len(components))
	}

	row, ok := components[0].(discordgo.ActionsRow)
	if !ok {
		t.Fatal("Expected ActionsRow component")
	}

	if len(row.Components) != 1 {
		t.Fatalf("Expected 1 button, got %d", len(row.Components))
	}

	startButton, ok := row.Components[0].(discordgo.Button)
	if !ok || startButton.Label != "Start Game" {
		t.Error("Expected Start Game button")
	}
}

func TestStartGame(t *testing.T) {
	mock := &MockSession{}
	channelID := "test-channel"

	// Clear any existing games
	activeGames = make(map[string]*BlackjackGame)

	// Create initial game state
	deck, _ := cards.NewDeck()
	game := &BlackjackGame{
		Deck:      deck,
		GameState: Waiting,
		CreatorID: "user1",
		ChannelID: channelID,
	}
	activeGames[channelID] = game

	// Start the game
	interaction := createTestInteraction("user1", "Player1", channelID, "blackjack_start")
	HandleBlackjackButton(mock, interaction)

	if game.GameState != Playing {
		t.Error("Game should be in Playing state")
	}

	if len(game.PlayerHand) != 2 {
		t.Errorf("Expected 2 cards for player, got %d", len(game.PlayerHand))
	}

	if len(game.DealerHand) != 1 {
		t.Errorf("Expected 1 card for dealer, got %d", len(game.DealerHand))
	}
}

func TestGamePlay(t *testing.T) {
	mock := &MockSession{}
	channelID := "test-channel"

	// Clear any existing games
	activeGames = make(map[string]*BlackjackGame)

	// Setup game with known cards for testing
	deck, _ := cards.NewDeck()
	game := &BlackjackGame{
		Deck:      deck,
		GameState: Playing,
		CreatorID: "user1",
		ChannelID: channelID,
	}

	// Deal initial cards
	game.PlayerHand = []cards.Card{{Rank: "10", Suit: "♠"}, {Rank: "7", Suit: "♥"}}
	game.DealerHand = []cards.Card{{Rank: "8", Suit: "♣"}}

	activeGames[channelID] = game

	// Test hit
	hitInteraction := createTestInteraction("user1", "Player1", channelID, "blackjack_hit")
	HandleBlackjackButton(mock, hitInteraction)

	if len(game.PlayerHand) != 3 {
		t.Errorf("Expected 3 cards after hit, got %d", len(game.PlayerHand))
	}

	// Test stand
	game.GameState = Playing // Reset state for stand test
	standInteraction := createTestInteraction("user1", "Player1", channelID, "blackjack_stand")
	HandleBlackjackButton(mock, standInteraction)

	if game.GameState != Finished {
		t.Error("Game should be finished after stand")
	}
	if len(game.DealerHand) < 2 {
		t.Error("Dealer should have drawn cards after stand")
	}
}

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		hand []cards.Card
		want int
	}{
		{
			hand: []cards.Card{{Rank: "A", Suit: "♠"}, {Rank: "K", Suit: "♥"}},
			want: 21,
		},
		{
			hand: []cards.Card{{Rank: "A", Suit: "♠"}, {Rank: "A", Suit: "♥"}},
			want: 12,
		},
		{
			hand: []cards.Card{{Rank: "K", Suit: "♠"}, {Rank: "Q", Suit: "♥"}, {Rank: "J", Suit: "♣"}},
			want: 30,
		},
		{
			hand: []cards.Card{{Rank: "A", Suit: "♠"}, {Rank: "5", Suit: "♥"}, {Rank: "7", Suit: "♣"}},
			want: 13,
		},
	}

	for _, tt := range tests {
		got := calculateScore(tt.hand)
		if got != tt.want {
			t.Errorf("calculateScore(%v) = %v, want %v", tt.hand, got, tt.want)
		}
	}
}
