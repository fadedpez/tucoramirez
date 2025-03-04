package games

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot/cards"
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
	if game.GameStarted {
		t.Error("Game should not be started yet")
	}
	if len(game.Players) != 1 {
		t.Errorf("Expected 1 player, got %d", len(game.Players))
	}
	if game.Players[0].ID != "user1" {
		t.Errorf("Expected player ID user1, got %s", game.Players[0].ID)
	}

	// Verify join/start buttons
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

	joinButton, ok := row.Components[0].(discordgo.Button)
	if !ok || joinButton.Label != "Join Game" {
		t.Error("Expected Join Game button")
	}

	startButton, ok := row.Components[1].(discordgo.Button)
	if !ok || startButton.Label != "Start Game" {
		t.Error("Expected Start Game button")
	}
}

func TestJoinGame(t *testing.T) {
	mock := &MockSession{}
	channelID := "test-channel"

	// Clear any existing games
	activeGames = make(map[string]*BlackjackGame)

	// Create initial game
	creator := createTestInteraction("user1", "Player1", channelID, "")
	StartBlackjackGame(mock, creator)

	// Try to join with a second player
	joiner := createTestInteraction("user2", "Player2", channelID, "blackjack_join")
	HandleBlackjackButton(mock, joiner)

	game := activeGames[channelID]
	if len(game.Players) != 2 {
		t.Errorf("Expected 2 players, got %d", len(game.Players))
	}

	// Try to join with the same player again
	HandleBlackjackButton(mock, joiner)
	if len(game.Players) != 2 {
		t.Errorf("Expected still 2 players, got %d", len(game.Players))
	}

	// Try to join with maximum players
	for i := 3; i <= maxPlayers+1; i++ {
		joiner := createTestInteraction(fmt.Sprintf("user%d", i), fmt.Sprintf("Player%d", i), channelID, "blackjack_join")
		HandleBlackjackButton(mock, joiner)
	}

	if len(game.Players) > maxPlayers {
		t.Errorf("Expected maximum %d players, got %d", maxPlayers, len(game.Players))
	}
}

func TestStartGame(t *testing.T) {
	mock := &MockSession{}
	channelID := "test-channel"

	// Clear any existing games
	activeGames = make(map[string]*BlackjackGame)

	// Create and join game
	creator := createTestInteraction("user1", "Player1", channelID, "")
	StartBlackjackGame(mock, creator)

	joiner := createTestInteraction("user2", "Player2", channelID, "blackjack_join")
	HandleBlackjackButton(mock, joiner)

	// Non-creator tries to start
	nonCreator := createTestInteraction("user2", "Player2", channelID, "blackjack_start")
	HandleBlackjackButton(mock, nonCreator)

	game := activeGames[channelID]
	if game.GameStarted {
		t.Error("Game should not have started from non-creator")
	}

	// Creator starts game
	creatorStart := createTestInteraction("user1", "Player1", channelID, "blackjack_start")
	HandleBlackjackButton(mock, creatorStart)

	if !game.GameStarted {
		t.Error("Game should have started")
	}

	// Verify initial hands
	for _, player := range game.Players {
		if len(player.Hand) != 2 {
			t.Errorf("Expected 2 cards for player %s, got %d", player.Username, len(player.Hand))
		}
	}

	if len(game.DealerHand) != 2 {
		t.Errorf("Expected 2 cards for dealer, got %d", len(game.DealerHand))
	}
}

func TestGamePlay(t *testing.T) {
	mock := &MockSession{}
	channelID := "test-channel"

	// Clear any existing games
	activeGames = make(map[string]*BlackjackGame)

	// Setup and start game
	creator := createTestInteraction("user1", "Player1", channelID, "")
	StartBlackjackGame(mock, creator)

	joiner := createTestInteraction("user2", "Player2", channelID, "blackjack_join")
	HandleBlackjackButton(mock, joiner)

	creatorStart := createTestInteraction("user1", "Player1", channelID, "blackjack_start")
	HandleBlackjackButton(mock, creatorStart)

	game := activeGames[channelID]

	// Test hitting
	hit := createTestInteraction("user1", "Player1", channelID, "blackjack_hit")
	HandleBlackjackButton(mock, hit)

	player1 := game.Players[0]
	if len(player1.Hand) != 3 {
		t.Errorf("Expected 3 cards after hit, got %d", len(player1.Hand))
	}

	// Test standing
	stand := createTestInteraction("user2", "Player2", channelID, "blackjack_stand")
	HandleBlackjackButton(mock, stand)

	player2 := game.Players[1]
	if !player2.Stood {
		t.Error("Player 2 should be marked as stood")
	}

	// Test game completion
	stand1 := createTestInteraction("user1", "Player1", channelID, "blackjack_stand")
	HandleBlackjackButton(mock, stand1)

	if !game.GameOver {
		t.Error("Game should be over after all players stand")
	}

	if len(game.DealerHand) < 2 {
		t.Error("Dealer should have played their hand")
	}
}

func TestGameTimeout(t *testing.T) {
	mock := &MockSession{}
	channelID := "test-channel"

	// Clear any existing games
	activeGames = make(map[string]*BlackjackGame)

	// Create game
	creator := createTestInteraction("user1", "Player1", channelID, "")
	StartBlackjackGame(mock, creator)

	// Wait for timeout
	time.Sleep(joinTimeout + time.Second)

	// Verify game was cleaned up
	if _, exists := activeGames[channelID]; exists {
		t.Error("Game should have been cleaned up after timeout")
	}
}

func TestFormatGameState(t *testing.T) {
	game := &BlackjackGame{
		Players: []*Player{
			{
				Username: "Player1",
				Hand:     []cards.Card{{Rank: "A", Suit: "♠"}, {Rank: "K", Suit: "♥"}},
				Stood:    true,
			},
			{
				Username: "Player2",
				Hand:     []cards.Card{{Rank: "8", Suit: "♣"}, {Rank: "9", Suit: "♦"}, {Rank: "5", Suit: "♠"}},
				Busted:   true,
			},
		},
		DealerHand:  []cards.Card{{Rank: "J", Suit: "♠"}, {Rank: "7", Suit: "♥"}},
		GameStarted: true,
	}

	// Test pre-game state
	preGame := &BlackjackGame{
		Players: []*Player{
			{Username: "Player1"},
			{Username: "Player2"},
		},
		GameStarted: false,
	}
	preGameState := formatGameState(preGame, false)
	if !strings.Contains(preGameState, "Players (2/6)") {
		t.Error("Pre-game state should show player count")
	}

	// Test hidden dealer card
	hiddenState := formatGameState(game, false)
	if !strings.Contains(hiddenState, "[Hidden Card]") {
		t.Error("Dealer's second card should be hidden")
	}

	// Test revealed state
	revealedState := formatGameState(game, true)
	if !strings.Contains(revealedState, "J♠") || !strings.Contains(revealedState, "7♥") {
		t.Error("Dealer's cards should be revealed")
	}

	// Test player status indicators
	if !strings.Contains(revealedState, "🛑 STAND") {
		t.Error("Should show stand indicator")
	}
	if !strings.Contains(revealedState, "🚫 BUST") {
		t.Error("Should show bust indicator")
	}
}

func TestCalculateScore(t *testing.T) {
	tests := []struct {
		name     string
		hand     []cards.Card
		expected int
	}{
		{
			name: "Basic hand",
			hand: []cards.Card{
				{Rank: "K", Suit: "♠"},
				{Rank: "7", Suit: "♥"},
			},
			expected: 17,
		},
		{
			name: "Ace as 11",
			hand: []cards.Card{
				{Rank: "A", Suit: "♠"},
				{Rank: "7", Suit: "♥"},
			},
			expected: 18,
		},
		{
			name: "Ace as 1",
			hand: []cards.Card{
				{Rank: "A", Suit: "♠"},
				{Rank: "K", Suit: "♥"},
				{Rank: "7", Suit: "♣"},
			},
			expected: 18,
		},
		{
			name: "Multiple aces",
			hand: []cards.Card{
				{Rank: "A", Suit: "♠"},
				{Rank: "A", Suit: "♥"},
				{Rank: "A", Suit: "♣"},
			},
			expected: 13,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateScore(tt.hand)
			if score != tt.expected {
				t.Errorf("Expected score %d, got %d", tt.expected, score)
			}
		})
	}
}

func TestHandleJoin_AlreadyInGame(t *testing.T) {
	// Create a mock session
	mock := &MockSession{}
	
	// Create a game with one player
	game := &BlackjackGame{
		Players: []*Player{
			{
				ID:       "123",
				Username: "TestUser",
			},
		},
	}
	
	// Create an interaction from the same player
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID:       "123",
					Username: "TestUser",
				},
			},
		},
	}
	
	// Try to join again
	HandleBlackjackButton(mock, i)
	
	// Verify response
	if mock.interactionResponse == nil {
		t.Fatal("Expected interaction response, got nil")
	}
	
	// Verify it's an ephemeral message
	if mock.interactionResponse.Type != discordgo.InteractionResponseChannelMessageWithSource {
		t.Errorf("Expected response type %d, got %d",
			discordgo.InteractionResponseChannelMessageWithSource,
			mock.interactionResponse.Type)
	}
	
	if mock.interactionResponse.Data.Flags != discordgo.MessageFlagsEphemeral {
		t.Error("Expected ephemeral message flag")
	}
	
	if !strings.Contains(mock.interactionResponse.Data.Content, "already in this game") {
		t.Errorf("Expected 'already in game' message, got: %s", mock.interactionResponse.Data.Content)
	}
	
	// Verify game state wasn't changed
	if len(game.Players) != 1 {
		t.Errorf("Expected 1 player, got %d", len(game.Players))
	}
}
