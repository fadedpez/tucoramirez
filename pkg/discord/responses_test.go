package discord

import (
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/fadedpez/tucoramirez/pkg/services/blackjack"
)

// mockSession implements the SessionInterface for testing
type mockSession struct{}

// GuildMember returns a mock member for testing
func (m *mockSession) GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error) {
	return &discordgo.Member{
		User: &discordgo.User{
			Username: "TestPlayer-" + userID,
		},
	}, nil
}

// TestBlackjackScoring tests that the Discord layer correctly handles blackjack scoring
// specifically that a natural blackjack (2 cards totaling 21) beats a non-blackjack 21
func TestBlackjackScoring(t *testing.T) {
	// Create a mock session
	mockS := &mockSession{}

	// Create a game where player has a blackjack and dealer has 21 with 3 cards
	game := &blackjack.Game{
		Dealer: &blackjack.Hand{
			Cards: []*entities.Card{
				{Rank: entities.Ten, Suit: entities.Spades},  // 10
				{Rank: entities.Six, Suit: entities.Hearts}, // 6
				{Rank: entities.Five, Suit: entities.Clubs}, // 5
			},
			Status: blackjack.StatusStand,
		},
		Players: map[string]*blackjack.Hand{
			"player1": {
				Cards: []*entities.Card{
					{Rank: entities.Ace, Suit: entities.Spades},  // 11
					{Rank: entities.Ten, Suit: entities.Hearts}, // 10
				},
				Status: blackjack.StatusStand,
			},
		},
	}

	// Get the game results description
	result := getGameResultsDescription(game, mockS, "guild123")

	// Check that player1 is marked as a winner
	if !strings.Contains(result, "GANADOR") {
		t.Errorf("Expected player with blackjack to be marked as GANADOR, but got: %s", result)
	}

	// Check that the result contains the correct scores
	if !strings.Contains(result, "(21)") {
		t.Errorf("Expected result to contain player score of 21, but got: %s", result)
	}

	// Verify the dealer's score is also shown as 21
	if !strings.Contains(result, "El Dealer tiene 21") {
		t.Errorf("Expected result to mention dealer score of 21, but got: %s", result)
	}
}
