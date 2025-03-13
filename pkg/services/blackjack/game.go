package blackjack

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/fadedpez/tucoramirez/pkg/repositories/game"
)

var (
	ErrGameNotStarted = errors.New("game not started")
	ErrGameInProgress = errors.New("game already in progress")
	ErrPlayerNotFound = errors.New("player not found")
	ErrInvalidAction  = errors.New("invalid action for current game state")
)

// BlackjackDetails contains game-specific result details
type BlackjackDetails struct {
	DealerScore int
	IsBlackjack bool
	IsBust      bool
}

func (d *BlackjackDetails) GameType() entities.GameState {
	return entities.StateDealing // we'll need to add a GameTypeBlackjack constant
}

func (d *BlackjackDetails) ValidateDetails() error {
	if d.DealerScore < 0 || d.DealerScore > 31 {
		return errors.New("invalid dealer score")
	}
	return nil
}

type Game struct {
	ID        string
	State     entities.GameState
	Deck      *entities.Deck
	Players   map[string]*Hand // PlayerID -> Hand
	Dealer    *Hand
	shuffled  bool // Flag to track if the deck has been shuffled
	ChannelID string
	repo      game.Repository
}

func NewGame(channelID string, repo game.Repository) *Game {
	return &Game{
		State:     entities.StateWaiting,
		Players:   make(map[string]*Hand),
		Dealer:    NewHand(),
		ChannelID: channelID,
		repo:      repo,
	}
}

// AddPlayer adds a player to the game
func (g *Game) AddPlayer(playerID string) error {
	if g.State != entities.StateWaiting {
		return ErrGameInProgress
	}
	g.Players[playerID] = NewHand()
	return nil
}

// Start initializes the game with a fresh deck and deals initial cards
func (g *Game) Start() error {
	if g.State != entities.StateWaiting {
		return ErrGameInProgress
	}
	if len(g.Players) == 0 {
		return errors.New("no players in game")
	}

	// Try to load existing deck from repository
	log.Printf("Attempting to load deck for channel %s", g.ChannelID)
	deck, err := g.repo.GetDeck(context.Background(), g.ChannelID)
	if err != nil {
		log.Printf("Error loading deck: %v", err)
		return fmt.Errorf("failed to load deck: %v", err)
	}

	// Create a new deck if none exists
	if deck == nil {
		log.Printf("No existing deck found, creating new deck for channel %s", g.ChannelID)
		g.Deck = entities.NewDeck()
		g.Deck.Shuffle()

		// Save the new deck to the repository
		log.Printf("Saving new deck to repository for channel %s", g.ChannelID)
		if err := g.repo.SaveDeck(context.Background(), g.ChannelID, g.Deck.Cards); err != nil {
			log.Printf("Error saving deck: %v", err)
			return fmt.Errorf("failed to save deck: %v", err)
		}
	} else {
		log.Printf("Using existing deck with %d cards for channel %s", len(deck), g.ChannelID)
		g.Deck = &entities.Deck{Cards: deck}
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

	// Save the deck state after dealing
	if err := g.repo.SaveDeck(context.Background(), g.ChannelID, g.Deck.Cards); err != nil {
		return fmt.Errorf("failed to save deck: %v", err)
	}

	g.State = entities.StatePlaying
	return nil
}

// Hit adds a card to the player's hand
func (g *Game) Hit(playerID string) error {
	if g.State != entities.StatePlaying {
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
	if g.State != entities.StatePlaying {
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
	if g.State != entities.StatePlaying && g.State != entities.StateDealer {
		return ErrInvalidAction
	}

	g.State = entities.StateDealer

	// Dealer must hit on 16 and below, stand on 17 and above
	for GetBestScore(g.Dealer.Cards) < 17 {
		card := g.Deck.Draw()
		if card == nil {
			g.Deck = NewBlackjackDeck()
			g.shuffled = true
			card = g.Deck.Draw()
		}
		if err := g.Dealer.AddCard(card); err != nil {
			return err
		}
	}

	g.State = entities.StateComplete
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
	if g.State != entities.StateComplete {
		return nil, ErrInvalidAction
	}

	results := make([]HandResult, 0, len(g.Players))
	playerResults := make([]*entities.PlayerResult, 0, len(g.Players))
	dealerBJ := IsBlackjack(g.Dealer.Cards)
	dealerScore := GetBestScore(g.Dealer.Cards)

	for playerID, hand := range g.Players {
		result := HandResult{PlayerID: playerID}

		// Handle busted hands first
		if IsBust(hand.Cards) {
			result.Result = ResultLose
			result.Score = GetBestScore(hand.Cards)
			results = append(results, result)

			// Add player result
			playerResults = append(playerResults, &entities.PlayerResult{
				PlayerID: playerID,
				Result:   entities.Result(result.Result),
				Score:    result.Score,
			})
			continue
		}

		// Check for player blackjack
		if IsBlackjack(hand.Cards) {
			if dealerBJ {
				result.Result = ResultPush
			} else {
				result.Result = ResultBlackjack
			}
			result.Score = GetBestScore(hand.Cards)
			results = append(results, result)

			// Add player result
			playerResults = append(playerResults, &entities.PlayerResult{
				PlayerID: playerID,
				Result:   entities.Result(result.Result),
				Score:    result.Score,
			})
			continue
		}

		// Compare scores for non-blackjack hands
		playerScore := GetBestScore(hand.Cards)
		result.Score = playerScore

		if dealerBJ {
			result.Result = ResultLose
		} else if IsBust(g.Dealer.Cards) {
			result.Result = ResultWin
		} else if playerScore > dealerScore {
			result.Result = ResultWin
		} else if playerScore < dealerScore {
			result.Result = ResultLose
		} else {
			result.Result = ResultPush
		}

		results = append(results, result)

		// Add player result
		playerResults = append(playerResults, &entities.PlayerResult{
			PlayerID: playerID,
			Result:   entities.Result(result.Result),
			Score:    result.Score,
		})
	}

	gameResult := &entities.GameResult{
		ChannelID:     g.ChannelID,
		GameType:      entities.StateDealing,
		CompletedAt:   time.Now(),
		PlayerResults: playerResults,
		Details: &BlackjackDetails{
			DealerScore: dealerScore,
			IsBlackjack: dealerBJ,
			IsBust:      IsBust(g.Dealer.Cards),
		},
	}

	if err := g.repo.SaveGameResult(context.Background(), gameResult); err != nil {
		log.Printf("Failed to save game result: %v", err)
	}

	// Save final deck state for next game
	if err := g.repo.SaveDeck(context.Background(), g.ChannelID, g.Deck.Cards); err != nil {
		log.Printf("Failed to save final deck state: %v", err)
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

// WasShuffled returns true if the deck was shuffled during the last operation
func (g *Game) WasShuffled() bool {
	wasShuffled := g.shuffled
	g.shuffled = false // Reset the flag
	return wasShuffled
}
