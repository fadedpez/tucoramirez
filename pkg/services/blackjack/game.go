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

	// Betting fields
	Bets                 map[string]int64 // PlayerID -> Bet amount
	CurrentBettingPlayer int              // Index into PlayerOrder for whose turn it is to bet
	PayoutsProcessed     bool             // Flag to track if payouts have been processed

	// Turn tracking
	PlayerOrder []string // Ordered list of player IDs
	CurrentTurn int      // Index into PlayerOrder
}

func NewGame(channelID string, repo game.Repository) *Game {
	return &Game{
		State:     entities.StateWaiting,
		Players:   make(map[string]*Hand),
		Dealer:    NewHand(),
		ChannelID: channelID,
		repo:      repo,
		Bets:      make(map[string]int64),
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
	// Allow starting from either WAITING or BETTING state
	if g.State != entities.StateWaiting && g.State != entities.StateBetting {
		return ErrGameInProgress
	}
	if len(g.Players) == 0 {
		return errors.New("no players in game")
	}

	// If we're in WAITING state and not in BETTING, transition to BETTING first
	if g.State == entities.StateWaiting {
		// Set up player order for betting
		g.PlayerOrder = make([]string, 0, len(g.Players))
		for playerID := range g.Players {
			g.PlayerOrder = append(g.PlayerOrder, playerID)
		}

		// Initialize betting turn to the first player
		g.CurrentBettingPlayer = 0

		g.State = entities.StateBetting
		return nil
	}

	// Check if all players have placed bets when coming from BETTING state
	if !g.CheckAllBetsPlaced() {
		return errors.New("not all players have placed bets")
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

	// Set up player order
	g.PlayerOrder = make([]string, 0, len(g.Players))
	for playerID := range g.Players {
		g.PlayerOrder = append(g.PlayerOrder, playerID)
	}
	g.CurrentTurn = 0

	g.State = entities.StatePlaying
	return nil
}

// StartBetting transitions the game to the betting phase
func (g *Game) StartBetting() error {
	if g.State != entities.StateWaiting {
		return ErrInvalidAction
	}

	// Set up player order for betting
	g.PlayerOrder = make([]string, 0, len(g.Players))
	for playerID := range g.Players {
		g.PlayerOrder = append(g.PlayerOrder, playerID)
	}

	// Initialize betting turn to the first player
	g.CurrentBettingPlayer = 0

	g.State = entities.StateBetting
	return nil
}

// PlaceBet places a bet for a player
func (g *Game) PlaceBet(playerID string, amount int64) error {
	if g.State != entities.StateBetting {
		return ErrInvalidAction
	}

	// Check if player exists
	_, exists := g.Players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	// Check if it's this player's turn to bet
	// Only enforce turn order if PlayerOrder is initialized and has elements
	if len(g.PlayerOrder) > 0 {
		// Make sure CurrentBettingPlayer is within bounds
		if g.CurrentBettingPlayer >= len(g.PlayerOrder) {
			g.CurrentBettingPlayer = 0 // Reset to first player if out of bounds
		}

		// Check if it's this player's turn
		if g.PlayerOrder[g.CurrentBettingPlayer] != playerID {
			return errors.New("not your turn to bet")
		}
	} else {
		// If PlayerOrder is not initialized, initialize it now
		g.PlayerOrder = make([]string, 0, len(g.Players))
		for pid := range g.Players {
			g.PlayerOrder = append(g.PlayerOrder, pid)
		}

		// Find this player's position in the order
		for i, pid := range g.PlayerOrder {
			if pid == playerID {
				g.CurrentBettingPlayer = i
				break
			}
		}
	}

	// Store bet amount
	g.Bets[playerID] = amount

	// Move to next player's turn
	g.CurrentBettingPlayer++

	// Check if all players have placed bets
	if g.CheckAllBetsPlaced() {
		// If all players have placed bets, transition to dealing
		err := g.StartDealing()
		if err != nil {
			log.Printf("Error transitioning to dealing phase: %v", err)
			return err
		}
		// Immediately transition to playing state as well
		err = g.StartPlaying()
		if err != nil {
			log.Printf("Error transitioning to playing phase: %v", err)
			return err
		}
		return nil
	} else if g.CurrentBettingPlayer >= len(g.PlayerOrder) {
		// Reset to first player if we've gone through all players but not all have placed bets
		g.CurrentBettingPlayer = 0
	}

	return nil
}

// PlaceBetWithWalletUpdate places a bet for a player and updates their wallet
// Returns whether a loan was given and any error
func (g *Game) PlaceBetWithWalletUpdate(ctx context.Context, playerID string, betAmount int64, walletService WalletService) (bool, error) {
	// First validate the bet using the existing PlaceBet method
	err := g.PlaceBet(playerID, betAmount)
	if err != nil {
		return false, err
	}

	// Check if player has enough funds and give loan if needed
	wallet, loanGiven, err := walletService.EnsureFundsWithLoan(
		ctx,
		playerID,
		betAmount,
		100, // Standard loan amount of $100
	)
	if err != nil {
		// Revert the bet since the wallet update failed
		delete(g.Bets, playerID)
		// Move back to previous player's turn
		if g.CurrentBettingPlayer > 0 {
			g.CurrentBettingPlayer--
		}
		return false, fmt.Errorf("error ensuring funds: %w", err)
	}

	// Check if player has enough funds after potential loan
	if wallet.Balance < betAmount {
		// Revert the bet since player still doesn't have enough funds
		delete(g.Bets, playerID)
		// Move back to previous player's turn
		if g.CurrentBettingPlayer > 0 {
			g.CurrentBettingPlayer--
		}
		return loanGiven, fmt.Errorf("insufficient funds even after loan")
	}

	// Deduct from wallet
	err = walletService.RemoveFunds(
		ctx,
		playerID,
		betAmount,
		fmt.Sprintf("Blackjack bet"),
	)
	if err != nil {
		// Revert the bet since the wallet update failed
		delete(g.Bets, playerID)
		// Move back to previous player's turn
		if g.CurrentBettingPlayer > 0 {
			g.CurrentBettingPlayer--
		}
		return loanGiven, fmt.Errorf("error updating wallet: %w", err)
	}

	return loanGiven, nil
}

// CheckAllBetsPlaced returns true if all players have placed bets
func (g *Game) CheckAllBetsPlaced() bool {
	for playerID := range g.Players {
		if _, hasBet := g.Bets[playerID]; !hasBet {
			return false
		}
	}

	return true
}

// GetPlayerBet returns a player's current bet amount
func (g *Game) GetPlayerBet(playerID string) int64 {
	return g.Bets[playerID]
}

// StartDealing transitions the game from betting to dealing phase
func (g *Game) StartDealing() error {
	if g.State != entities.StateBetting {
		return ErrInvalidAction
	}

	// Check if all players have placed bets
	if !g.CheckAllBetsPlaced() {
		return errors.New("not all players have placed bets")
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

	// Set up player order for playing phase (reuse the same order from betting)
	g.CurrentTurn = 0

	// Transition to DEALING state
	g.State = entities.StateDealing

	// We'll return nil immediately to allow the UI to update with the DEALING state
	return nil
}

// StartPlaying transitions the game from dealing to playing phase
func (g *Game) StartPlaying() error {
	if g.State != entities.StateDealing {
		return ErrInvalidAction
	}

	// Transition to PLAYING state
	g.State = entities.StatePlaying
	return nil
}

// Hit adds a card to the player's hand
func (g *Game) Hit(playerID string) error {
	if g.State != entities.StatePlaying {
		return ErrInvalidAction
	}

	// Check if it's this player's turn
	if !g.IsPlayerTurn(playerID) {
		return errors.New("not your turn")
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
		// If the player busts or has another error, advance to the next player's turn
		if err == ErrHandBust {
			g.AdvanceTurn()
		}
		return err
	}

	return nil
}

// Stand marks the player's hand as complete
func (g *Game) Stand(playerID string) error {
	if g.State != entities.StatePlaying {
		return ErrInvalidAction
	}

	// Check if it's this player's turn
	if !g.IsPlayerTurn(playerID) {
		return errors.New("not your turn")
	}

	hand, exists := g.Players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	err := hand.Stand()
	if err == nil {
		// Advance to next player's turn
		g.AdvanceTurn()
	}

	return err
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

	// Transition to complete state
	g.State = entities.StateComplete

	return nil
}

// FinishGame completes the game and processes payouts
// This should be called after the dealer has played or all players have busted
func (g *Game) FinishGame(ctx context.Context, walletService WalletService) error {
	// Ensure the game is in complete state
	if g.State != entities.StateComplete {
		g.State = entities.StateComplete
	}

	// Process payouts
	return g.ProcessPayouts(ctx, walletService)
}

// ProcessPayouts processes payouts and updates player wallets
func (g *Game) ProcessPayouts(ctx context.Context, walletService WalletService) error {
	log.Printf("[DEBUG] Starting payout processing for game in channel %s", g.ChannelID)

	// Ensure game is in the COMPLETE state
	if g.State != entities.StateComplete {
		log.Printf("Cannot process payouts for game in state %s", g.State)
		return fmt.Errorf("invalid action for current game state")
	}

	// Check if payouts have already been processed
	if g.PayoutsProcessed {
		log.Printf("Payouts already processed for this game, skipping")
		return nil
	}

	log.Printf("[DEBUG] Game state: %s, Players: %d, Bets: %d", g.State, len(g.Players), len(g.Bets))

	// Calculate payouts
	payouts := g.CalculatePayouts()

	// Process each player's payout
	for playerID, payout := range payouts {
		log.Printf("Processing payout for player %s: $%d", playerID, payout)

		// Get player's current wallet
		log.Printf("[DEBUG] Getting wallet for player %s", playerID)
		wallet, created, err := walletService.GetOrCreateWallet(ctx, playerID)
		if err != nil {
			log.Printf("Error getting wallet for player %s: %v", playerID, err)
			continue
		}

		log.Printf("Before payout: Player %s wallet balance: $%d (wallet was just created: %v)", playerID, wallet.Balance, created)

		// Add winnings to wallet if there are any
		if payout > 0 {
			log.Printf("Adding $%d winnings to player %s wallet with description: Blackjack winnings", payout, playerID)
			log.Printf("[DEBUG] Calling walletService.AddFunds for player %s, amount %d", playerID, payout)
			
			err = walletService.AddFunds(ctx, playerID, payout, "Blackjack winnings")
			if err != nil {
				log.Printf("Error adding winnings to player %s wallet: %v", playerID, err)
				continue
			}
			log.Printf("Successfully added $%d winnings to player %s wallet", payout, playerID)

			// Get updated wallet to verify
			log.Printf("[DEBUG] Getting updated wallet for player %s after payout", playerID)
			updatedWallet, _, err := walletService.GetOrCreateWallet(ctx, playerID)
			if err != nil {
				log.Printf("Error getting updated wallet for player %s: %v", playerID, err)
			} else {
				log.Printf("After payout: Player %s wallet balance: $%d (expected: $%d)", playerID, updatedWallet.Balance, wallet.Balance+payout)
			}
		} else {
			log.Printf("Player %s has zero payout (likely lost)", playerID)
		}
	}

	// Mark payouts as processed
	g.PayoutsProcessed = true
	log.Printf("Payouts processed and marked as complete")

	return nil
}

// CalculatePayouts calculates the payout amounts for each player based on their results
// This method does not update any wallets, it just calculates the payout amounts
func (g *Game) CalculatePayouts() map[string]int64 {
	// Calculate payouts for each player
	payouts := make(map[string]int64)
	
	// Get the results
	results, err := g.GetResults()
	if err != nil {
		log.Printf("Error getting results for payouts: %v", err)
		return payouts
	}

	// Calculate payout for each player
	for _, result := range results {
		bet := g.Bets[result.PlayerID]

		switch result.Result {
		case ResultWin:
			// Regular win pays 1:1 (original bet + equal amount as winnings)
			// Since we already deducted the bet when it was placed, we need to return
			// the original bet plus the winnings
			payouts[result.PlayerID] = bet * 2
		case ResultBlackjack:
			// Blackjack pays 3:2 (original bet + 1.5x bet as winnings)
			// Since we already deducted the bet when it was placed, we need to return
			// the original bet plus the winnings
			payouts[result.PlayerID] = bet + (bet * 3 / 2)
		case ResultPush:
			// Push returns the original bet
			payouts[result.PlayerID] = bet
		case ResultLose:
			// Loss pays nothing (bet was already deducted when placed)
			payouts[result.PlayerID] = 0
		}

		log.Printf("Player %s bet $%d, result: %s, payout: $%d",
			result.PlayerID, bet, result.Result, payouts[result.PlayerID])
	}
	log.Printf("Calculated payouts: %v", payouts)

	// Check if any players are missing from the payouts
	for playerID := range g.Bets {
		if _, exists := payouts[playerID]; !exists {
			log.Printf("WARNING: Player %s has a bet but no payout calculated", playerID)
		}
	}

	return payouts
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

// GetCurrentTurnPlayerID returns the ID of the player whose turn it is
func (g *Game) GetCurrentTurnPlayerID() (string, error) {
	if g.State != entities.StatePlaying {
		return "", ErrInvalidAction
	}
	if len(g.PlayerOrder) == 0 {
		return "", errors.New("no players in game")
	}
	return g.PlayerOrder[g.CurrentTurn], nil
}

// AdvanceTurn moves to the next player's turn
func (g *Game) AdvanceTurn() {
	g.CurrentTurn = (g.CurrentTurn + 1) % len(g.PlayerOrder)

	// Skip players who are done (bust or stand)
	for g.CheckPlayerDone(g.PlayerOrder[g.CurrentTurn]) && !g.CheckAllPlayersDone() {
		g.CurrentTurn = (g.CurrentTurn + 1) % len(g.PlayerOrder)
	}
}

// IsPlayerTurn checks if it's the specified player's turn
func (g *Game) IsPlayerTurn(playerID string) bool {
	currentPlayer, err := g.GetCurrentTurnPlayerID()
	if err != nil {
		return false
	}
	return playerID == currentPlayer
}

// WalletService defines the interface for wallet operations
type WalletService interface {
	GetOrCreateWallet(ctx context.Context, userID string) (*entities.Wallet, bool, error)
	AddFunds(ctx context.Context, userID string, amount int64, description string) error
	RemoveFunds(ctx context.Context, userID string, amount int64, description string) error
	EnsureFundsWithLoan(ctx context.Context, userID string, requiredAmount int64, loanAmount int64) (*entities.Wallet, bool, error)
}

// GetPlayerWallets retrieves wallets for all players in the game and identifies the highest balance
// Returns a map of player IDs to wallets and the highest balance amount
func (g *Game) GetPlayerWallets(ctx context.Context, walletService WalletService) (map[string]*entities.Wallet, int64, error) {
	playerWallets := make(map[string]*entities.Wallet)
	highestBalance := int64(-1)

	// Use PlayerOrder if available, otherwise use the Players map
	playerIDs := g.PlayerOrder
	if len(playerIDs) == 0 {
		playerIDs = make([]string, 0, len(g.Players))
		for playerID := range g.Players {
			playerIDs = append(playerIDs, playerID)
		}
	}

	// Collect all player wallets
	for _, playerID := range playerIDs {
		wallet, _, err := walletService.GetOrCreateWallet(ctx, playerID)
		if err != nil {
			log.Printf("Error getting wallet for player %s: %v", playerID, err)
			continue
		}

		playerWallets[playerID] = wallet
		if wallet.Balance > highestBalance {
			highestBalance = wallet.Balance
		}
	}

	return playerWallets, highestBalance, nil
}
