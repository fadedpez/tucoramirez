package blackjack

import (
	"fmt"
	"testing"

	"github.com/bwmarrin/discordgo"
	discordmock "github.com/fadedpez/tucoramirez/internal/discord/mock"
	"github.com/fadedpez/tucoramirez/pkg/cards"
	"github.com/fadedpez/tucoramirez/pkg/games/common"
	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/mock"
)

type GameSuite struct {
	suite.Suite
	game    *Game
	session *discordmock.SessionHandler
}

func (s *GameSuite) SetupTest() {
	s.session = &discordmock.SessionHandler{}
	s.session.Test(s.T())
	s.game = NewGame("creator123", "channel456")
	s.game.Deck = cards.NewDeck() // Initialize the deck

	// Set up default mock responses
	s.session.On("InteractionRespond", mock.Anything, mock.Anything).Return(nil)
}

func TestGameSuite(t *testing.T) {
	suite.Run(t, new(GameSuite))
}

func (s *GameSuite) TestNewGame() {
	s.Equal("creator123", s.game.CreatorID)
	s.Equal("channel456", s.game.ChannelID)
	s.Equal(StateWaiting, s.game.State)
	s.NotNil(s.game.Players)
	s.Empty(s.game.Players)
}

func (s *GameSuite) TestAddPlayer() {
	player := common.NewPlayer("player123")
	err := s.game.AddPlayer(player)
	s.NoError(err)
	s.Contains(s.game.Players, "player123")
	s.Equal(player, s.game.Players["player123"])
}

func (s *GameSuite) TestAddPlayerWhenGameFull() {
	// Fill the game with max players
	for i := 0; i < MaxPlayers; i++ {
		player := common.NewPlayer(fmt.Sprintf("player%d", i))
		err := s.game.AddPlayer(player)
		s.NoError(err)
	}

	// Try to add one more player
	player := common.NewPlayer("extraPlayer")
	err := s.game.AddPlayer(player)
	s.Error(err)
	s.NotContains(s.game.Players, "extraPlayer")
}

func (s *GameSuite) TestAddPlayerWhenAlreadyInGame() {
	player := common.NewPlayer("player123")
	err := s.game.AddPlayer(player)
	s.NoError(err)

	// Try to add the same player again
	err = s.game.AddPlayer(player)
	s.Error(err)
	s.Len(s.game.Players, 1)
}

func (s *GameSuite) TestStart() {
	// Add a player
	player := common.NewPlayer("player123")
	s.game.AddPlayer(player)

	err := s.game.Start()
	s.NoError(err)
	s.Equal(StatePlaying, s.game.State)
	s.NotNil(s.game.Deck)
	s.Len(s.game.DealerHand, 2)
	s.Len(player.Hand, 2)
}

func (s *GameSuite) TestStartWithNoPlayers() {
	err := s.game.Start()
	s.Error(err)
	s.Equal(StateWaiting, s.game.State)
}

func (s *GameSuite) TestHandleHit() {
	// Setup game with a player
	player := common.NewPlayer("player123")
	s.game.AddPlayer(player)
	s.game.Start()

	// Mock Discord interaction
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID: "player123",
				},
			},
		},
	}

	// Set up mock expectations
	s.session.On("InteractionRespond", mock.Anything, mock.MatchedBy(func(r *discordgo.InteractionResponse) bool {
		return r.Type == discordgo.InteractionResponseUpdateMessage
	})).Return(nil)

	// Execute hit
	err := s.game.HandleHit(s.session, i)

	// Assert results
	s.NoError(err)
	s.Len(player.Hand, 3) // Initial 2 cards + 1 from hit
	s.session.AssertExpectations(s.T())
}

func (s *GameSuite) TestHandleStand() {
	// Setup game with a player
	player := common.NewPlayer("player123")
	s.game.AddPlayer(player)
	s.game.Start()

	// Mock Discord interaction
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{
				User: &discordgo.User{
					ID: "player123",
				},
			},
		},
	}

	// Set up mock expectations
	s.session.On("InteractionRespond", mock.Anything, mock.MatchedBy(func(r *discordgo.InteractionResponse) bool {
		return r.Type == discordgo.InteractionResponseUpdateMessage
	})).Return(nil)

	// Execute stand
	err := s.game.HandleStand(s.session, i)

	// Assert results
	s.NoError(err)
	s.True(player.Stood)
	s.session.AssertExpectations(s.T())
}

func (s *GameSuite) TestCalculateScore() {
	hand := []cards.Card{
		{Rank: cards.Ace, Suit: cards.Hearts},
		{Rank: cards.King, Suit: cards.Spades},
	}
	score := calculateScore(hand)
	s.Equal(21, score)

	// Test with multiple aces
	hand = []cards.Card{
		{Rank: cards.Ace, Suit: cards.Hearts},
		{Rank: cards.Ace, Suit: cards.Spades},
		{Rank: cards.Nine, Suit: cards.Diamonds},
	}
	score = calculateScore(hand)
	s.Equal(21, score)
}

func (s *GameSuite) TestIsFinished() {
	s.False(s.game.IsFinished())
	s.game.State = StateFinished
	s.True(s.game.IsFinished())
}

func (s *GameSuite) TestGetButtons() {
	// Test waiting state buttons
	s.game.State = StateWaiting
	buttons := s.game.GetButtons()
	s.NotNil(buttons)
	s.Len(buttons, 1) // One action row
	actionRow := buttons[0].(discordgo.ActionsRow)
	s.Len(actionRow.Components, 2) // Join and Start buttons
	s.Equal("Join Game", actionRow.Components[0].(discordgo.Button).Label)
	s.Equal("Start Game", actionRow.Components[1].(discordgo.Button).Label)

	// Test playing state buttons
	s.game.State = StatePlaying
	buttons = s.game.GetButtons()
	s.NotNil(buttons)
	s.Len(buttons, 1) // One action row
	actionRow = buttons[0].(discordgo.ActionsRow)
	s.Len(actionRow.Components, 2) // Hit and Stand buttons
	s.Equal("Hit", actionRow.Components[0].(discordgo.Button).Label)
	s.Equal("Stand", actionRow.Components[1].(discordgo.Button).Label)

	// Test finished state buttons
	s.game.State = StateFinished
	buttons = s.game.GetButtons()
	s.NotNil(buttons)
	s.Len(buttons, 1) // One action row
	actionRow = buttons[0].(discordgo.ActionsRow)
	s.Len(actionRow.Components, 1) // Play Again button
	s.Equal("Play Again", actionRow.Components[0].(discordgo.Button).Label)
}

func (s *GameSuite) TestString() {
	// Add a player and start the game
	player := common.NewPlayer("player123")
	s.game.AddPlayer(player)
	s.game.Start()
	player.Username = "player123"

	str := s.game.String()
	s.Contains(str, "Dealer's Hand")
	s.Contains(str, "player123")
}
