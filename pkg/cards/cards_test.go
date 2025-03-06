package cards

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type CardsTestSuite struct {
	suite.Suite
}

func TestCardsSuite(t *testing.T) {
	suite.Run(t, new(CardsTestSuite))
}

func (s *CardsTestSuite) TestCardString() {
	testCases := []struct {
		name     string
		card     Card
		expected string
	}{
		{
			name:     "ace of hearts",
			card:     Card{Suit: Hearts, Rank: Ace},
			expected: "♥A",
		},
		{
			name:     "ten of diamonds",
			card:     Card{Suit: Diamonds, Rank: Ten},
			expected: "♦10",
		},
		{
			name:     "king of clubs",
			card:     Card{Suit: Clubs, Rank: King},
			expected: "♣K",
		},
		{
			name:     "queen of spades",
			card:     Card{Suit: Spades, Rank: Queen},
			expected: "♠Q",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Execute
			result := tc.card.String()

			// Assert
			s.Equal(tc.expected, result, "Card string representation should match expected")
		})
	}
}

func (s *CardsTestSuite) TestNewDeck() {
	// Execute
	deck := NewDeck()

	// Assert
	s.NotNil(deck, "Deck should not be nil")
	s.Len(deck.Cards, 52, "Deck should have 52 cards")

	// Verify all suits and ranks are present
	suits := map[Suit]int{Hearts: 0, Diamonds: 0, Clubs: 0, Spades: 0}
	ranks := map[Rank]int{
		Ace: 0, Two: 0, Three: 0, Four: 0, Five: 0,
		Six: 0, Seven: 0, Eight: 0, Nine: 0, Ten: 0,
		Jack: 0, Queen: 0, King: 0,
	}

	for _, card := range deck.Cards {
		suits[card.Suit]++
		ranks[card.Rank]++
	}

	for suit, count := range suits {
		s.Equal(13, count, "Each suit should have 13 cards: %s", suit)
	}

	for rank, count := range ranks {
		s.Equal(4, count, "Each rank should have 4 cards: %s", rank)
	}
}

func (s *CardsTestSuite) TestShuffle() {
	// Setup
	deck1 := NewDeck()
	deck2 := NewDeck()

	// Verify decks are initially in the same order
	for i := range deck1.Cards {
		s.Equal(deck1.Cards[i], deck2.Cards[i], "Initial decks should be in the same order")
	}

	// Execute
	deck1.Shuffle()

	// Assert
	cardsMatch := true
	for i := range deck1.Cards {
		if deck1.Cards[i] != deck2.Cards[i] {
			cardsMatch = false
			break
		}
	}
	s.False(cardsMatch, "Shuffled deck should be in different order than original")

	// Verify no cards were lost or duplicated
	s.Len(deck1.Cards, 52, "Shuffled deck should still have 52 cards")
	cardCounts := make(map[Card]int)
	for _, card := range deck1.Cards {
		cardCounts[card]++
	}
	for card, count := range cardCounts {
		s.Equal(1, count, "Card %v should appear exactly once", card)
	}
}

func (s *CardsTestSuite) TestDraw() {
	testCases := []struct {
		name          string
		drawCount     int
		expectedDraw  int
		expectedRemain int
	}{
		{
			name:          "draw zero cards",
			drawCount:     0,
			expectedDraw:  0,
			expectedRemain: 52,
		},
		{
			name:          "draw one card",
			drawCount:     1,
			expectedDraw:  1,
			expectedRemain: 51,
		},
		{
			name:          "draw multiple cards",
			drawCount:     5,
			expectedDraw:  5,
			expectedRemain: 47,
		},
		{
			name:          "draw all cards",
			drawCount:     52,
			expectedDraw:  52,
			expectedRemain: 0,
		},
		{
			name:          "draw more than deck size",
			drawCount:     60,
			expectedDraw:  52,
			expectedRemain: 0,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Setup
			deck := NewDeck()

			// Execute
			drawn := deck.Draw(tc.drawCount)

			// Assert
			s.Len(drawn, tc.expectedDraw, "Should draw expected number of cards")
			s.Len(deck.Cards, tc.expectedRemain, "Deck should have expected number of remaining cards")

			// Verify drawn cards are valid
			for _, card := range drawn {
				s.NotEmpty(card.Suit, "Drawn card should have a suit")
				s.NotEmpty(card.Rank, "Drawn card should have a rank")
			}
		})
	}
}

func (s *CardsTestSuite) TestDrawOne() {
	// Setup
	deck := NewDeck()
	initialCard := deck.Cards[0]

	// Execute
	drawn := deck.DrawOne()

	// Assert
	s.Equal(initialCard, drawn, "DrawOne should return the top card")
	s.Len(deck.Cards, 51, "Deck should have one less card")

	// Test drawing from empty deck
	emptyDeck := &Deck{}
	emptyCard := emptyDeck.DrawOne()
	s.Empty(emptyCard.Suit, "Drawing from empty deck should return empty card")
	s.Empty(emptyCard.Rank, "Drawing from empty deck should return empty card")
}
