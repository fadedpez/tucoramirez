package games

import (
	"testing"

	"github.com/fadedpez/tucoramirez/tucobot/cards"
)

func TestNewGame(t *testing.T) {
	game := NewGame()

	if game == nil {
		t.Fatal("Expected game to not be nil")
	}

	if len(game.Deck) != 52 {
		t.Errorf("Expected deck to have 52 cards, got %d", len(game.Deck))
	}

	if game.Dealer == nil {
		t.Error("Expected dealer to not be nil")
	}

	if len(game.Players) != 0 {
		t.Errorf("Expected no players initially, got %d", len(game.Players))
	}
}

func TestUpdatePlayerScore(t *testing.T) {
	tests := []struct {
		name     string
		hand     []cards.Card
		expected int
	}{
		{
			name: "Basic hand",
			hand: []cards.Card{
				{Value: ":keycap_ten:", Suit: ":hearts:"},
				{Value: ":seven:", Suit: ":clubs:"},
			},
			expected: 17,
		},
		{
			name: "Blackjack",
			hand: []cards.Card{
				{Value: ":regional_indicator_a:", Suit: ":hearts:"},
				{Value: ":regional_indicator_k:", Suit: ":clubs:"},
			},
			expected: 21,
		},
		{
			name: "Multiple aces",
			hand: []cards.Card{
				{Value: ":regional_indicator_a:", Suit: ":hearts:"},
				{Value: ":regional_indicator_a:", Suit: ":clubs:"},
				{Value: ":nine:", Suit: ":diamonds:"},
			},
			expected: 21,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := &cards.Player{
				ID:   "test",
				Name: "Test Player",
				Hand: tt.hand,
			}
			updatePlayerScore(player)
			if player.Score != tt.expected {
				t.Errorf("Expected score %d, got %d", tt.expected, player.Score)
			}
		})
	}
}

func TestDealCard(t *testing.T) {
	gameSession = NewGame()
	initialDeckSize := len(gameSession.Deck)
	
	player := &cards.Player{
		ID:   "test",
		Name: "Test Player",
		Hand: []cards.Card{},
	}

	dealCard(player)

	if len(gameSession.Deck) != initialDeckSize-1 {
		t.Errorf("Expected deck size to decrease by 1, got %d", len(gameSession.Deck))
	}

	if len(player.Hand) != 1 {
		t.Errorf("Expected player to have 1 card, got %d", len(player.Hand))
	}
}

func TestDetermineWinner(t *testing.T) {
	tests := []struct {
		name         string
		playerScore  int
		dealerScore  int
		expectedWins bool
	}{
		{"Player busts", 22, 20, false},
		{"Dealer busts", 20, 22, true},
		{"Player wins", 20, 18, true},
		{"Dealer wins", 18, 20, false},
		{"Tie", 20, 20, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gameSession = NewGame()
			player := &cards.Player{
				ID:    "test",
				Name:  "Test Player",
				Score: tt.playerScore,
			}
			gameSession.Dealer.Score = tt.dealerScore

			result := determineWinner(player)
			containsWins := result == "Test Player wins!" || result == "Dealer busted! Test Player wins!"
			
			if containsWins != tt.expectedWins {
				t.Errorf("Expected wins=%v, got result=%s", tt.expectedWins, result)
			}
		})
	}
}

func TestGetGameState(t *testing.T) {
	gameSession = NewGame()
	
	// Add a test player
	gameSession.Players = append(gameSession.Players, cards.Player{
		ID:   "test",
		Name: "Test Player",
		Hand: []cards.Card{
			{Value: ":keycap_ten:", Suit: ":hearts:"},
			{Value: ":seven:", Suit: ":clubs:"},
		},
		Score: 17,
	})

	// Add dealer cards
	gameSession.Dealer.Hand = []cards.Card{
		{Value: ":nine:", Suit: ":diamonds:"},
		{Value: ":eight:", Suit: ":spades:"},
	}
	gameSession.Dealer.Score = 17

	state := getGameState()
	
	if state == "" {
		t.Error("Expected non-empty game state")
	}
}
