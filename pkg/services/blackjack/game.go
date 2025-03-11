package blackjack

import (
	"errors"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

var (
	ErrGameNotStarted = errors.New("game not started")
	ErrGameInProgress = errors.New("game already in progress")
	ErrPlayerNotFound = errors.New("player not found")
	ErrInvalidAction  = errors.New("invalid action for current game state")
)

type GameState string

const (
	StateWaiting  GameState = "WAITING"  // Waiting for players
	StateDealing  GameState = "DEALING"  // Initial deal in progress
	StatePlaying  GameState = "PLAYING"  // Players taking turns
	StateDealer   GameState = "DEALER"   // Dealer's turn
	StateComplete GameState = "COMPLETE" // Round complete
)

type Game struct {
	ID        string
	State     GameState
	Deck      *entities.Deck
	Players   map[string]*Hand // PlayerID -> Hand
	Dealer    *Hand
	shuffled  bool // Flag to track if the deck has been shuffled
	ChannelID string
}

func NewGame(channelID string) *Game {
	return &Game{
		State:     StateWaiting,
		Players:   make(map[string]*Hand),
		Dealer:    NewHand(),
		ChannelID: channelID,
	}
}

// AddPlayer adds a player to the game
func (g *Game) AddPlayer(playerID string) error {
	if g.State != StateWaiting {
		return ErrGameInProgress
	}
	g.Players[playerID] = NewHand()
	return nil
}

// Start initializes the game with a fresh deck and deals initial cards
func (g *Game) Start() error {
	if g.State != StateWaiting {
		return ErrGameInProgress
	}
	if len(g.Players) == 0 {
		return errors.New("no players in game")
	}

	// Only create a new deck if we don't have one or we need to reshuffle
	if g.Deck == nil || ShouldReshuffle(g.Deck) {
		g.Deck = NewBlackjackDeck()
		g.shuffled = true
	}

	// Deal initial cards
	for i := 0; i < 2; i++ {
		// Deal to each player
		for _, hand := range g.Players {
			if card := g.Deck.Draw(); card != nil {
				hand.AddCard(card)
			}
		}
		// Deal to dealer
		if card := g.Deck.Draw(); card != nil {
			g.Dealer.AddCard(card)
		}
	}

	g.State = StatePlaying
	return nil
}

// Hit adds a card to the player's hand
func (g *Game) Hit(playerID string) error {
	if g.State != StatePlaying {
		return ErrInvalidAction
	}

	hand, exists := g.Players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	// Draw and add card
	card := g.Deck.Draw()
	if card == nil {
		// Only create a new deck if we're completely out of cards
		g.Deck = NewBlackjackDeck()
		g.shuffled = true
		card = g.Deck.Draw()
	}

	if err := hand.AddCard(card); err != nil {
		return err
	}

	return nil
}

// Stand marks the player's hand as complete
func (g *Game) Stand(playerID string) error {
	if g.State != StatePlaying {
		return ErrInvalidAction
	}

	hand, exists := g.Players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	return hand.Stand()
}

// PlayDealer executes dealer's turn according to house rules
func (g *Game) PlayDealer() error {
	if g.State != StatePlaying && g.State != StateDealer {
		return ErrInvalidAction
	}

	g.State = StateDealer

	// Dealer must hit on 16 and below, stand on 17 and above
	for GetBestScore(g.Dealer.Cards) < 17 {
		card := g.Deck.Draw()
		if card == nil {
			return errors.New("no cards remaining")
		}
		if err := g.Dealer.AddCard(card); err != nil {
			return err
		}
	}

	g.State = StateComplete
	return nil
}

// Result represents the outcome of a hand
type Result string

const (
	ResultWin       Result = "WIN"
	ResultLose      Result = "LOSE"
	ResultPush      Result = "PUSH"
	ResultBlackjack Result = "BLACKJACK"
)

// HandResult stores the result of a player's hand
type HandResult struct {
	PlayerID string
	Result   Result
	Score    int
}

// GetResults evaluates all hands against the dealer and returns results
func (g *Game) GetResults() ([]HandResult, error) {
	if g.State != StateComplete {
		return nil, ErrInvalidAction
	}

	results := make([]HandResult, 0, len(g.Players))
	dealerBJ := IsBlackjack(g.Dealer.Cards)

	for playerID, hand := range g.Players {
		result := HandResult{PlayerID: playerID}

		// Handle busted hands first
		if IsBust(hand.Cards) {
			result.Result = ResultLose
			result.Score = GetBestScore(hand.Cards)
			results = append(results, result)
			continue
		}

		// Check for player blackjack
		if IsBlackjack(hand.Cards) {
			if dealerBJ {
				result.Result = ResultPush
			} else {
				result.Result = ResultBlackjack
			}
			result.Score = 21
			results = append(results, result)
			continue
		}

		// Handle dealer bust
		if IsBust(g.Dealer.Cards) {
			result.Result = ResultWin
			result.Score = GetBestScore(hand.Cards)
			results = append(results, result)
			continue
		}

		// Compare scores
		playerScore := GetBestScore(hand.Cards)
		result.Score = playerScore

		switch CompareHands(hand.Cards, g.Dealer.Cards) {
		case 1:
			result.Result = ResultWin
		case -1:
			result.Result = ResultLose
		case 0:
			result.Result = ResultPush
		}

		results = append(results, result)
	}

	return results, nil
}

// CheckPlayerDone checks if a player is no longer able to take actions
func (g *Game) CheckPlayerDone(playerID string) bool {
	hand, exists := g.Players[playerID]
	if !exists {
		return true
	}
	return hand.Status != StatusPlaying
}

// CheckAllPlayersDone checks if all players have either bust or stood
func (g *Game) CheckAllPlayersDone() bool {
	for _, hand := range g.Players {
		if hand.Status == StatusPlaying {
			return false
		}
	}
	return true
}

// CheckAllPlayersBust checks if all players have bust
func (g *Game) CheckAllPlayersBust() bool {
	for _, hand := range g.Players {
		if hand.Status != StatusBust {
			return false
		}
	}
	return true
}
