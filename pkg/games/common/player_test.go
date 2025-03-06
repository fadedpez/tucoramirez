package common

import (
	"testing"

	"github.com/fadedpez/tucoramirez/pkg/cards"
	"github.com/stretchr/testify/suite"
)

type PlayerTestSuite struct {
	suite.Suite
	player *Player
}

func TestPlayerSuite(t *testing.T) {
	suite.Run(t, new(PlayerTestSuite))
}

func (s *PlayerTestSuite) SetupTest() {
	s.player = NewPlayer("test_player")
}

func (s *PlayerTestSuite) TestNewPlayer() {
	// Setup
	id := "test_id"

	// Execute
	player := NewPlayer(id)

	// Assert
	s.NotNil(player, "Player should not be nil")
	s.Equal(id, player.ID, "Player should have correct ID")
	s.Empty(player.Username, "New player should have empty username")
	s.Empty(player.Hand, "New player should have empty hand")
	s.Zero(player.Score, "New player should have zero score")
	s.False(player.Stood, "New player should not be standing")
	s.False(player.Busted, "New player should not be busted")
}

func (s *PlayerTestSuite) TestClearHand() {
	// Setup
	s.player.Hand = []cards.Card{{Suit: cards.Hearts, Rank: cards.Ace}}
	s.player.Score = 11
	s.player.Stood = true
	s.player.Busted = true

	// Execute
	s.player.ClearHand()

	// Assert
	s.Empty(s.player.Hand, "Hand should be empty after clearing")
	s.Zero(s.player.Score, "Score should be reset to zero")
	s.False(s.player.Stood, "Player should no longer be standing")
	s.False(s.player.Busted, "Player should no longer be busted")
}

func (s *PlayerTestSuite) TestAddCard() {
	// Setup
	card1 := cards.Card{Suit: cards.Hearts, Rank: cards.Ace}
	card2 := cards.Card{Suit: cards.Spades, Rank: cards.King}

	// Execute & Assert
	s.Empty(s.player.Hand, "Hand should start empty")

	s.player.AddCard(card1)
	s.Len(s.player.Hand, 1, "Hand should have one card")
	s.Equal(card1, s.player.Hand[0], "First card should match added card")

	s.player.AddCard(card2)
	s.Len(s.player.Hand, 2, "Hand should have two cards")
	s.Equal(card2, s.player.Hand[1], "Second card should match added card")
}

func (s *PlayerTestSuite) TestGetHand() {
	// Setup
	testCards := []cards.Card{
		{Suit: cards.Hearts, Rank: cards.Ace},
		{Suit: cards.Spades, Rank: cards.King},
	}
	s.player.Hand = testCards

	// Execute
	hand := s.player.GetHand()

	// Assert
	s.Equal(testCards, hand, "GetHand should return the current hand")
}

func (s *PlayerTestSuite) TestSetAndGetScore() {
	// Setup & Assert initial state
	s.Zero(s.player.GetScore(), "Initial score should be zero")

	// Execute & Assert
	testScores := []int{0, 10, 21, -5}
	for _, score := range testScores {
		s.player.SetScore(score)
		s.Equal(score, s.player.GetScore(), "Score should match set value")
	}
}

func (s *PlayerTestSuite) TestStandAndHasStood() {
	// Assert initial state
	s.False(s.player.HasStood(), "Player should not start standing")

	// Execute
	s.player.Stand()

	// Assert
	s.True(s.player.HasStood(), "Player should be standing after Stand()")
}

func (s *PlayerTestSuite) TestBustAndHasBusted() {
	// Assert initial state
	s.False(s.player.HasBusted(), "Player should not start busted")

	// Execute
	s.player.Bust()

	// Assert
	s.True(s.player.HasBusted(), "Player should be busted after Bust()")
}

func (s *PlayerTestSuite) TestPlayerStateIndependence() {
	// Setup
	player1 := NewPlayer("player1")
	player2 := NewPlayer("player2")

	// Execute different actions on each player
	player1.AddCard(cards.Card{Suit: cards.Hearts, Rank: cards.Ace})
	player1.SetScore(11)
	player1.Stand()

	player2.AddCard(cards.Card{Suit: cards.Spades, Rank: cards.King})
	player2.SetScore(10)
	player2.Bust()

	// Assert players maintain independent state
	s.NotEqual(player1.Hand, player2.Hand, "Players should have different hands")
	s.NotEqual(player1.Score, player2.Score, "Players should have different scores")
	s.True(player1.HasStood(), "Player 1 should be standing")
	s.False(player1.HasBusted(), "Player 1 should not be busted")
	s.False(player2.HasStood(), "Player 2 should not be standing")
	s.True(player2.HasBusted(), "Player 2 should be busted")
}
