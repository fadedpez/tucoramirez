package blackjack

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/fadedpez/tucoramirez/pkg/repositories/game"
)

// Game-specific errors
var (
	ErrGameNotStarted     = errors.New("game not started")
	ErrGameInProgress     = errors.New("game already in progress")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrInvalidAction      = errors.New("invalid action for current game state")
	ErrMaxPlayersReached  = errors.New("maximum number of players reached")
	ErrNoPlayers          = errors.New("no players in game")
	ErrNotAllBetsPlaced   = errors.New("not all players have placed bets")
	ErrFailedToLoadDeck   = errors.New("failed to load deck")
	ErrFailedToSaveDeck   = errors.New("failed to save deck")
	ErrInvalidBet         = errors.New("invalid bet amount")
	ErrNotPlayerTurn      = errors.New("not player's turn")
	ErrPlayerAlreadyBet   = errors.New("player already placed a bet")
	ErrInvalidDealerScore = errors.New("invalid dealer score")
	// Special betting errors
	ErrNotEligibleForDoubleDown   = errors.New("not eligible for double down")
	ErrNotEligibleForSplit        = errors.New("not eligible for split")
	ErrNotEligibleForInsurance    = errors.New("not eligible for insurance")
	ErrInsufficientFundsForAction = errors.New("insufficient funds for this action")
	ErrNotAllPlayersHaveBet       = errors.New("not all players have placed bets")
)

// GameType constants
const (
	StateBlackjack entities.GameState = "BLACKJACK"
	// Special betting states
	StateSplitting   entities.GameState = "SPLITTING"
	StateSpecialBets entities.GameState = "SPECIAL_BETS"
)

// BlackjackDetails contains game-specific result details
type BlackjackDetails struct {
	DealerScore int
	IsBlackjack bool
	IsBust      bool
}

func (d *BlackjackDetails) GameType() entities.GameState {
	return StateBlackjack
}

func (d *BlackjackDetails) ValidateDetails() error {
	if d.DealerScore < 0 || d.DealerScore > 31 {
		return ErrInvalidDealerScore
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

	// Special bets tracking
	CurrentSpecialBetsTurn int // Index into PlayerOrder for special bets
}

const StandardLoanAmount = 100

func NewGame(channelID string, repo game.Repository) *Game {
	return &Game{
		State:     entities.StateWaiting,
		Players:   make(map[string]*Hand),
		Dealer:    NewHand(),
		ChannelID: channelID,
		repo:      repo,
		Bets:      make(map[string]int64),
		Deck:      entities.NewDeck(), // Initialize with a new deck to avoid nil pointer issues
	}
}

// AddPlayer adds a player to the game
func (g *Game) AddPlayer(playerID string) error {
	if g.State != entities.StateWaiting {
		return ErrGameInProgress
	}

	// Check if maximum players limit has been reached
	if len(g.Players) >= 7 {
		return ErrMaxPlayersReached
	}

	g.Players[playerID] = NewHand()
	return nil
}

// Start initializes the game with a fresh deck and deals initial cards
func (g *Game) Start() error {
	// Allow starting from either WAITING or BETTING state
	if g.State != entities.StateWaiting && g.State != entities.StateBetting {
		return ErrInvalidAction
	}
	if len(g.Players) == 0 {
		return ErrNoPlayers
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
		return ErrNotAllBetsPlaced
	}

	// First transition to DEALING state
	log.Printf("Transitioning from BETTING to DEALING state")
	g.State = entities.StateDealing

	// Try to load existing deck from repository
	log.Printf("Attempting to load deck for channel %s", g.ChannelID)
	deck, err := g.repo.GetDeck(context.Background(), g.ChannelID)
	if err != nil {
		log.Printf("Error loading deck: %v", err)
		return ErrFailedToLoadDeck
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
			return ErrFailedToSaveDeck
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
		return ErrFailedToSaveDeck
	}

	// Set up player order
	g.PlayerOrder = make([]string, 0, len(g.Players))
	for playerID := range g.Players {
		g.PlayerOrder = append(g.PlayerOrder, playerID)
	}
	g.CurrentTurn = 0

	// Check for special betting options
	if g.AnyPlayerEligibleForSplit() {
		// Transition to splitting state
		log.Printf("Transitioning to splitting state as players are eligible for split")
		g.State = StateSplitting
		g.CurrentSpecialBetsTurn = 0
	} else if g.AnyPlayerEligibleForSpecialBets() {
		// Transition to special bets state
		log.Printf("Transitioning to special bets state as players are eligible for special bets")
		g.State = StateSpecialBets
		g.CurrentSpecialBetsTurn = 0
	} else {
		// No special bets available, go directly to playing state
		log.Printf("No special betting options available, transitioning directly to playing state")
		g.State = entities.StatePlaying
	}

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

// ValidateBet checks if a player can place a bet
func (g *Game) ValidateBet(playerID string) error {
	// Check if game is in betting state
	if g.State != entities.StateBetting {
		return ErrInvalidAction
	}

	// Check if player is in the game
	if _, exists := g.Players[playerID]; !exists {
		return ErrPlayerNotFound
	}

	// Check if it's this player's turn to bet
	if len(g.PlayerOrder) > 0 && g.CurrentBettingPlayer < len(g.PlayerOrder) {
		currentPlayerID := g.PlayerOrder[g.CurrentBettingPlayer]
		if currentPlayerID != playerID {
			return ErrNotPlayerTurn
		}
	}

	// Check if player has already bet
	if _, hasBet := g.Bets[playerID]; hasBet {
		return ErrPlayerAlreadyBet
	}

	return nil
}

// PlaceBet places a bet for a player
func (g *Game) PlaceBet(playerID string, amount int64) error {
	if err := g.ValidateBet(playerID); err != nil {
		return err
	}

	// Store bet amount
	g.Bets[playerID] = amount

	// Move to next player's turn
	g.CurrentBettingPlayer++

	// Check if all players have placed bets
	if g.CheckAllBetsPlaced() {
		// If all players have placed bets, transition to dealing and playing
		// using the Start method which properly handles deck initialization
		log.Printf("All bets placed, transitioning to dealing and playing phases")
		err := g.Start()
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

	// Get the standard loan increment from the service
	loanAmount := walletService.GetStandardLoanIncrement()

	// Check if player has enough funds and give loan if needed
	wallet, loanGiven, err := walletService.EnsureFundsWithLoan(
		ctx,
		playerID,
		betAmount,
		loanAmount, // Use the standard loan amount from the service
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
		"Blackjack bet",
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
	for _, playerID := range g.PlayerOrder {
		_, exists := g.Bets[playerID]
		if !exists {
			return ErrNotAllPlayersHaveBet
		}
	}

	// Safety check: ensure we have a valid deck before dealing
	if g.Deck == nil {
		log.Printf("Warning: Deck was nil in StartDealing, initializing new deck")
		// Try to load existing deck from repository
		if g.repo != nil {
			deck, err := g.repo.GetDeck(context.Background(), g.ChannelID)
			if err == nil && deck != nil {
				log.Printf("Loaded existing deck with %d cards from repository", len(deck))
				g.Deck = &entities.Deck{Cards: deck}
			} else {
				log.Printf("Could not load deck from repository, creating new deck")
				g.Deck = entities.NewDeck()
				g.Deck.Shuffle()
				g.shuffled = true
			}
		} else {
			log.Printf("Repository not available, creating new deck")
			g.Deck = entities.NewDeck()
			g.Deck.Shuffle()
			g.shuffled = true
		}
	}

	// Deal two cards to each player
	for _, playerID := range g.PlayerOrder {
		hand := g.Players[playerID]
		for i := 0; i < 2; i++ {
			card := g.Deck.Draw()
			if card == nil {
				// Reshuffle if deck is empty
				g.Deck = entities.NewDeck()
				g.Deck.Shuffle()
				g.shuffled = true
				card = g.Deck.Draw()
			}
			hand.AddCard(card)
		}
	}

	// Deal two cards to the dealer
	for i := 0; i < 2; i++ {
		card := g.Deck.Draw()
		if card == nil {
			g.Deck = entities.NewDeck()
			g.Deck.Shuffle()
			g.shuffled = true
			card = g.Deck.Draw()
		}
		g.Dealer.AddCard(card)
	}

	// Save the deck state after dealing if repository is available
	if g.repo != nil {
		if err := g.repo.SaveDeck(context.Background(), g.ChannelID, g.Deck.Cards); err != nil {
			log.Printf("Warning: Failed to save deck state: %v", err)
		}
	}

	// Check for special betting options
	if g.AnyPlayerEligibleForSplit() {
		// Transition to splitting state
		g.State = StateSplitting
		g.CurrentSpecialBetsTurn = 0
	} else if g.AnyPlayerEligibleForSpecialBets() {
		// Transition to special bets state
		g.State = StateSpecialBets
		g.CurrentSpecialBetsTurn = 0
	} else {
		// No special bets available, go directly to playing state
		g.State = entities.StatePlaying
		g.CurrentTurn = 0
	}

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
		return ErrNotPlayerTurn
	}

	// Get the current player ID from the player order
	currentPlayerID, err := g.GetCurrentTurnPlayerID()
	if err != nil {
		return err
	}

	// Determine which hand we're working with (original or split)
	targetHandID := currentPlayerID
	if strings.HasSuffix(currentPlayerID, "_split") && strings.TrimSuffix(currentPlayerID, "_split") == playerID {
		// If the current turn is for a split hand and the player ID matches the original player
		targetHandID = currentPlayerID
	}

	hand, exists := g.Players[targetHandID]
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

	// Check if player busted after adding the card
	if hand.Status == StatusBust {
		g.AdvanceTurn()
		
		// Log the transition for debugging
		log.Printf("Player %s busted on hand %s, advancing to next turn", playerID, targetHandID)
		
		// Check if all players are done and transition to dealer state if needed
		if g.CheckAllPlayersDone() {
			log.Printf("All players are done after bust, transitioning to dealer's turn")
			g.State = entities.StateDealer
		}
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
		return ErrNotPlayerTurn
	}

	// Get the current player ID from the player order
	currentPlayerID, err := g.GetCurrentTurnPlayerID()
	if err != nil {
		return err
	}

	// Determine which hand we're working with (original or split)
	targetHandID := currentPlayerID
	if strings.HasSuffix(currentPlayerID, "_split") && strings.TrimSuffix(currentPlayerID, "_split") == playerID {
		// If the current turn is for a split hand and the player ID matches the original player
		targetHandID = currentPlayerID
	}

	hand, exists := g.Players[targetHandID]
	if !exists {
		return ErrPlayerNotFound
	}

	err = hand.Stand()
	if err == nil {
		// Advance to next player's turn
		g.AdvanceTurn()

		// Log the transition for debugging
		log.Printf("Player %s stood on hand %s, advancing to next turn", playerID, targetHandID)

		// Check if all players are done and transition to dealer state if needed
		if g.CheckAllPlayersDone() {
			log.Printf("All players are done after stand, transitioning to dealer's turn")
			g.State = entities.StateDealer
		}
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
		err := g.Dealer.AddCard(card)
		if err != nil {
			return err // Any error here is a real error
		}

		// Check if dealer busted
		if g.Dealer.Status == StatusBust {
			break
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
		return ErrInvalidAction
	}

	// Check if payouts have already been processed
	if g.PayoutsProcessed {
		log.Printf("Payouts already processed for this game, skipping")
		return nil
	}

	log.Printf("[DEBUG] Game state: %s, Players: %d, Bets: %d", g.State, len(g.Players), len(g.Bets))

	// Get detailed results for game record
	handResults, err := g.GetResults()
	if err != nil {
		log.Printf("Error getting detailed results: %v", err)
		return err
	}

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

	// Save game record to repository if available
	if g.repo != nil {
		// Create game record
		gameRecord := game.GameRecord{
			ID:            g.ID,
			GameType:      "blackjack",
			ChannelID:     g.ChannelID,
			StartTime:     time.Now().Add(-30 * time.Minute), // Estimate start time
			EndTime:       time.Now(),
			PlayerRecords: make([]game.HandRecord, 0, len(handResults)),
			DealerCards:   getCardStrings(g.Dealer.Cards),
			DealerScore:   g.Dealer.Value(),
		}

		// Add hand records
		for _, result := range handResults {
			// Get cards for this hand
			var cards []string
			if hand, exists := g.Players[result.HandID]; exists {
				cards = getCardStrings(hand.Cards)
			}

			// Create hand record
			handRecord := game.HandRecord{
				PlayerID:        result.PlayerID,
				HandID:          result.HandID,
				ParentHandID:    result.ParentHandID,
				Cards:           cards,
				FinalScore:      result.Score,
				InitialBet:      result.Bet,
				IsSplit:         result.IsSplit,
				IsDoubledDown:   result.IsDoubledDown,
				DoubleDownBet:   result.DoubleDownBet,
				HasInsurance:    result.HasInsurance,
				InsuranceBet:    result.InsuranceBet,
				Result:          convertStringResultToResult(result.Result),
				Payout:          result.Payout,
				InsurancePayout: result.InsurancePayout,
				Actions:         []string{}, // We don't track individual actions yet
				Metadata:        make(map[string]interface{}),
			}

			// Add to game record
			gameRecord.PlayerRecords = append(gameRecord.PlayerRecords, handRecord)
		}

		// Save game record
		log.Printf("Saving game record to repository for game %s", g.ID)
		err := g.repo.SaveGameResult(ctx, convertToGameResult(gameRecord))
		if err != nil {
			log.Printf("Error saving game record: %v", err)
			// Continue processing - this is not a critical error
		} else {
			log.Printf("Successfully saved game record for game %s", g.ID)
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
		// Use the payout already calculated in GetResults
		payout := result.Payout

		// Add insurance payout if applicable
		if result.HasInsurance {
			payout += result.InsurancePayout
		}

		// Add the payout to the player's total
		if existingPayout, exists := payouts[result.PlayerID]; exists {
			payouts[result.PlayerID] = existingPayout + payout
		} else {
			payouts[result.PlayerID] = payout
		}

		// Log detailed payout information
		logMsg := fmt.Sprintf("Player %s hand %s bet $%d", result.PlayerID, result.HandID, result.Bet)
		if result.IsDoubledDown {
			logMsg += fmt.Sprintf(" (doubled down +$%d)", result.DoubleDownBet)
		}
		logMsg += fmt.Sprintf(", result: %s, payout: $%d", result.Result, payout)
		if result.HasInsurance {
			logMsg += fmt.Sprintf(" (includes insurance payout: $%d)", result.InsurancePayout)
		}
		log.Println(logMsg)
	}

	log.Printf("Calculated payouts: %v", payouts)

	// Check if any players are missing from the payouts
	for playerID := range g.Bets {
		// Skip split hand IDs (they contain an underscore)
		if strings.Contains(playerID, "_") {
			continue
		}

		if _, exists := payouts[playerID]; !exists {
			log.Printf("WARNING: Player %s has a bet but no payout calculated", playerID)
		}
	}

	return payouts
}

// CalculateResults calculates the results of the game
func (g *Game) CalculateResults(ctx context.Context, walletService WalletService) (map[string]*entities.PlayerResult, error) {
	results := make(map[string]*entities.PlayerResult)

	// Calculate dealer's best score
	dealerScore := GetBestScore(g.Dealer.Cards)

	// Check if dealer busted
	dealerBusted := dealerScore > 21

	// Check if dealer has blackjack (21 with 2 cards)
	dealerBlackjack := dealerScore == 21 && len(g.Dealer.Cards) == 2

	// Process each player's result
	for playerID, hand := range g.Players {
		// Skip split hands for now, we'll handle them separately
		if hand.IsSplit() && hand.GetParentHandID() != "" {
			continue
		}

		// Get the player's bet
		betAmount, exists := g.Bets[playerID]
		if !exists {
			return nil, fmt.Errorf("no bet found for player %s", playerID)
		}

		// Calculate player's best score
		playerScore := hand.Value()

		// Check if player busted
		playerBusted := playerScore > 21

		// Check if player has blackjack (21 with 2 cards)
		playerBlackjack := playerScore == 21 && len(hand.Cards) == 2 && !hand.IsSplit()

		// Initialize result
		result := &entities.PlayerResult{
			PlayerID: playerID,
			Score:    playerScore,
		}

		// Calculate payout based on result
		payout := int64(0)
		if playerBusted {
			// Player busts, loses bet
			result.Result = entities.StringResultLose
			payout = 0
		} else if dealerBusted {
			// Dealer busts, player wins
			result.Result = entities.StringResultWin

			// Check if player doubled down
			if hand.IsDoubledDown() {
				// Double down wins pay 2:1 on the total bet
				doubleDownBet := hand.GetDoubleDownBet()
				totalBet := betAmount + doubleDownBet
				payout = totalBet * 2
			} else {
				// Normal wins pay 1:1
				payout = betAmount * 2
			}
		} else if playerBlackjack && !dealerBlackjack {
			// Player has blackjack, dealer doesn't
			result.Result = entities.StringResultBlackjack
			// Blackjack pays 3:2
			payout = betAmount + int64(float64(betAmount)*1.5)
		} else if dealerBlackjack && !playerBlackjack {
			// Dealer has blackjack, player doesn't
			result.Result = entities.StringResultLose
			payout = 0

			// Check for insurance
			if hand.HasInsurance() {
				insuranceBet := hand.GetInsuranceBet()
				// Insurance pays 2:1 when dealer has blackjack
				payout = insuranceBet * 3
			}
		} else if dealerBlackjack && playerBlackjack {
			// Both have blackjack, it's a push
			result.Result = entities.StringResultPush
			payout = betAmount

			// Check for insurance
			if hand.HasInsurance() {
				insuranceBet := hand.GetInsuranceBet()
				// Insurance pays 2:1 when dealer has blackjack
				payout += insuranceBet * 3
			}
		} else if playerScore > dealerScore {
			// Player score is higher than dealer
			result.Result = entities.StringResultWin

			// Check if player doubled down
			if hand.IsDoubledDown() {
				// Double down wins pay 2:1 on the total bet
				doubleDownBet := hand.GetDoubleDownBet()
				totalBet := betAmount + doubleDownBet
				payout = totalBet * 2
			} else {
				// Normal wins pay 1:1
				payout = betAmount * 2
			}
		} else if playerScore < dealerScore {
			// Dealer score is higher than player
			result.Result = entities.StringResultLose
			payout = 0

			// Check for insurance if dealer has blackjack
			if dealerBlackjack && hand.HasInsurance() {
				insuranceBet := hand.GetInsuranceBet()
				// Insurance pays 2:1 when dealer has blackjack
				payout = insuranceBet * 3
			}
		} else {
			// Scores are equal, it's a push
			result.Result = entities.StringResultPush
			payout = betAmount
		}

		// Process the payout
		if payout > 0 {
			// Add the winnings to the player's wallet
			err := walletService.AddFunds(
				ctx,
				playerID,
				payout,
				"Blackjack winnings",
			)
			if err != nil {
				return nil, err
			}
		}

		// Store the result
		results[playerID] = result

		// Handle split hands if present
		if hand.IsSplit() && hand.GetSplitHandID() != "" {
			splitHandID := hand.GetSplitHandID()
			splitHand, exists := g.Players[splitHandID]

			if exists {
				// Get the split hand bet
				splitBetAmount, exists := g.Bets[splitHandID]
				if !exists {
					return nil, fmt.Errorf("no bet found for split hand %s", splitHandID)
				}

				// Calculate split hand score
				splitScore := splitHand.Value()

				// Check if split hand busted
				splitBusted := splitScore > 21

				// Initialize split result
				splitResult := &entities.PlayerResult{
					PlayerID: splitHandID,
					Score:    splitScore,
				}

				// Calculate payout for split hand
				splitPayout := int64(0)
				if splitBusted {
					// Split hand busts, loses bet
					splitResult.Result = entities.StringResultLose
					splitPayout = 0
				} else if dealerBusted {
					// Dealer busts, split hand wins
					splitResult.Result = entities.StringResultWin
					splitPayout = splitBetAmount * 2
				} else if splitScore > dealerScore {
					// Split hand score is higher than dealer
					splitResult.Result = entities.StringResultWin
					splitPayout = splitBetAmount * 2
				} else if splitScore < dealerScore {
					// Dealer score is higher than split hand
					splitResult.Result = entities.StringResultLose
					splitPayout = 0
				} else {
					// Scores are equal, it's a push
					splitResult.Result = entities.StringResultPush
					splitPayout = splitBetAmount
				}

				// Process the payout for the split hand
				if splitPayout > 0 {
					// Add the winnings to the player's wallet
					err := walletService.AddFunds(
						ctx,
						playerID, // Note: winnings go to the original player, not the split hand ID
						splitPayout,
						"Blackjack split hand winnings",
					)
					if err != nil {
						return nil, err
					}
				}

				// Store the split result
				results[splitHandID] = splitResult
			}
		}
	}

	return results, nil
}

// ProcessResults processes the results of the game and returns the payouts
func (g *Game) ProcessResults(ctx context.Context, walletService WalletService) (map[string]int64, error) {
	if g.State != entities.StateComplete {
		return nil, ErrInvalidAction
	}

	// Calculate results
	results, err := g.CalculateResults(ctx, walletService)
	if err != nil {
		return nil, err
	}

	// Convert to payouts map
	payouts := make(map[string]int64)
	for playerID, result := range results {
		// For now, we're not tracking the actual payout amount in the result
		// This will be enhanced in future versions
		if result.Result.IsWin() {
			payouts[playerID] = g.Bets[playerID] * 2 // Simple 1:1 payout
		} else if result.Result == entities.StringResultPush {
			payouts[playerID] = g.Bets[playerID] // Return original bet
		}
	}

	return payouts, nil
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
		return "", ErrNoPlayers
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

	// Direct match for the player ID
	if playerID == currentPlayer {
		return true
	}

	// Check if this is a split hand situation
	// If the current player is a split hand (e.g., "player1_split"),
	// then the original player ("player1") should be able to play it
	if strings.HasSuffix(currentPlayer, "_split") {
		originalPlayer := strings.TrimSuffix(currentPlayer, "_split")
		return playerID == originalPlayer
	}

	return false
}

// IsGameComplete returns true if the game is in a completed state
func (g *Game) IsGameComplete() bool {
	return g.State == entities.StateComplete || g.CheckAllPlayersDone()
}

// PlayerInfo contains all UI-relevant information about a player
type PlayerInfo struct {
	PlayerID          string
	HasBet            bool
	BetAmount         int64
	WalletBalance     int64
	HasHighestBalance bool
	IsCurrentTurn     bool
}

// GameUIInfo contains all UI-relevant information about the game
type GameUIInfo struct {
	CurrentPlayerInfo    *PlayerInfo
	AllPlayersInfo       []PlayerInfo
	WasShuffled          bool
	ShuffleMessage       string
	ShouldProcessPayouts bool
}

// WalletService defines the interface for wallet operations
type WalletService interface {
	GetOrCreateWallet(ctx context.Context, userID string) (*entities.Wallet, bool, error)
	AddFunds(ctx context.Context, userID string, amount int64, description string) error
	RemoveFunds(ctx context.Context, userID string, amount int64, description string) error
	EnsureFundsWithLoan(ctx context.Context, userID string, requiredAmount int64, loanAmount int64) (*entities.Wallet, bool, error)
	GetStandardLoanIncrement() int64
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

// GetAllPlayersInfo returns information about all players in the game
func (g *Game) GetAllPlayersInfo(ctx context.Context, walletService WalletService) ([]PlayerInfo, error) {
	playerWallets, highestBalance, err := g.GetPlayerWallets(ctx, walletService)
	if err != nil {
		return nil, err
	}

	var playersInfo []PlayerInfo

	// Use PlayerOrder if available
	if len(g.PlayerOrder) > 0 {
		for i, playerID := range g.PlayerOrder {
			wallet := playerWallets[playerID]
			bet, hasBet := g.Bets[playerID]

			playersInfo = append(playersInfo, PlayerInfo{
				PlayerID:          playerID,
				HasBet:            hasBet,
				BetAmount:         bet,
				WalletBalance:     wallet.Balance,
				HasHighestBalance: wallet.Balance == highestBalance,
				IsCurrentTurn:     g.CurrentBettingPlayer == i,
			})
		}
	} else {
		// Fallback to Players map
		for playerID := range g.Players {
			wallet := playerWallets[playerID]
			bet, hasBet := g.Bets[playerID]

			playersInfo = append(playersInfo, PlayerInfo{
				PlayerID:          playerID,
				HasBet:            hasBet,
				BetAmount:         bet,
				WalletBalance:     wallet.Balance,
				HasHighestBalance: wallet.Balance == highestBalance,
				IsCurrentTurn:     false, // Can't determine current turn without PlayerOrder
			})
		}
	}

	return playersInfo, nil
}

// CompleteGameIfDone checks if the game is complete, plays the dealer if needed,
// processes payouts, and returns whether the game is complete
func (g *Game) CompleteGameIfDone(ctx context.Context, walletService WalletService) (bool, error) {
	// If the game is already complete, no need to check again
	if g.State == entities.StateComplete {
		return true, nil
	}

	// If the game is in the DEALER state, handle dealer play and complete the game
	if g.State == entities.StateDealer {
		log.Printf("Game is in DEALER state, handling dealer play and completing game")

		// Check if all players have busted
		allPlayersBusted := true
		for _, hand := range g.Players {
			if hand.Status != StatusBust {
				allPlayersBusted = false
				break
			}
		}

		// If all players busted, dealer automatically stands without drawing
		if !allPlayersBusted {
			// Play the dealer's turn only if at least one player hasn't busted
			if err := g.PlayDealer(); err != nil {
				// If the error is because the dealer busted, that's a valid game state
				if err == ErrHandBust {
					log.Printf("Dealer busted, continuing with game completion")
				} else {
					return false, err
				}
			}
		} else {
			log.Printf("All players busted, dealer automatically stands")
		}

		// Transition to COMPLETE state
		log.Printf("Dealer's turn complete, transitioning to COMPLETE state")
		g.State = entities.StateComplete

		// Process payouts
		if err := g.FinishGame(ctx, walletService); err != nil {
			return true, err
		}

		return true, nil
	}

	// If the game is not in the PLAYING state, it's not ready to be completed
	if g.State != entities.StatePlaying {
		return false, nil
	}

	// Check if all players are done (bust or stand)
	allDone := g.CheckAllPlayersDone()
	log.Printf("Checking if all players are done: %v", allDone)
	if !allDone {
		return false, nil
	}

	// Transition to dealer state
	log.Printf("All players are done, transitioning to dealer's turn")
	g.State = entities.StateDealer

	// We've transitioned to dealer state, but we'll let the next call handle the dealer's play
	// This ensures the UI can update to show the dealer state before proceeding
	return false, nil
}

// GetCurrentBettingPlayerInfo returns information about the current betting player
func (g *Game) GetCurrentBettingPlayerInfo(ctx context.Context, walletService WalletService) (*PlayerInfo, error) {
	if g.CurrentBettingPlayer >= len(g.PlayerOrder) {
		return nil, nil
	}

	playerID := g.PlayerOrder[g.CurrentBettingPlayer]
	wallet, _, err := walletService.GetOrCreateWallet(ctx, playerID)
	if err != nil {
		return nil, err
	}

	bet, hasBet := g.Bets[playerID]

	return &PlayerInfo{
		PlayerID:      playerID,
		HasBet:        hasBet,
		BetAmount:     bet,
		WalletBalance: wallet.Balance,
		IsCurrentTurn: true,
	}, nil
}

// GetCurrentPlayerInfo returns information about the current player whose turn it is to play
func (g *Game) GetCurrentPlayerInfo(ctx context.Context, walletService WalletService) (*PlayerInfo, error) {
	if g.State != entities.StatePlaying || g.CurrentTurn >= len(g.PlayerOrder) {
		return nil, nil
	}

	playerID := g.PlayerOrder[g.CurrentTurn]
	wallet, _, err := walletService.GetOrCreateWallet(ctx, playerID)
	if err != nil {
		return nil, err
	}

	bet, hasBet := g.Bets[playerID]

	return &PlayerInfo{
		PlayerID:      playerID,
		HasBet:        hasBet,
		BetAmount:     bet,
		WalletBalance: wallet.Balance,
		IsCurrentTurn: true,
	}, nil
}

// GetShuffleInfo returns information about whether the deck was shuffled
func (g *Game) GetShuffleInfo() (bool, string) {
	if g.shuffled {
		g.shuffled = false // Reset after checking
		return true, "We've been playing a long time eh my friends? Let Tuco shuffle the deck, maybe it bring Tuco more luck."
	}
	return false, ""
}

// ShouldProcessPayouts returns whether payouts should be processed
func (g *Game) ShouldProcessPayouts() bool {
	return g.State == entities.StateComplete && !g.PayoutsProcessed
}

// GetGameUIInfo returns all UI-relevant information about the game
func (g *Game) GetGameUIInfo(ctx context.Context, walletService WalletService) (*GameUIInfo, error) {
	var currentPlayer *PlayerInfo
	var err error

	// Get appropriate current player based on game state
	if g.State == entities.StateBetting {
		currentPlayer, err = g.GetCurrentBettingPlayerInfo(ctx, walletService)
		if err != nil {
			return nil, err
		}
	} else if g.State == entities.StatePlaying {
		currentPlayer, err = g.GetCurrentPlayerInfo(ctx, walletService)
		if err != nil {
			return nil, err
		}
	}

	allPlayers, err := g.GetAllPlayersInfo(ctx, walletService)
	if err != nil {
		return nil, err
	}

	wasShuffled, shuffleMessage := g.GetShuffleInfo()

	return &GameUIInfo{
		CurrentPlayerInfo:    currentPlayer,
		AllPlayersInfo:       allPlayers,
		WasShuffled:          wasShuffled,
		ShuffleMessage:       shuffleMessage,
		ShouldProcessPayouts: g.ShouldProcessPayouts(),
	}, nil
}

// HandResult stores the result of a player's hand
type HandResult struct {
	PlayerID        string
	HandID          string
	Result          entities.StringResult
	Score           int
	Bet             int64
	Payout          int64
	IsSplit         bool
	ParentHandID    string
	IsDoubledDown   bool
	DoubleDownBet   int64
	HasInsurance    bool
	InsuranceBet    int64
	InsurancePayout int64
}

// GetResults evaluates all hands against the dealer and returns results
// This method is maintained for backward compatibility with tests
func (g *Game) GetResults() ([]HandResult, error) {
	if g.State != entities.StateComplete {
		return nil, ErrInvalidAction
	}

	results := make([]HandResult, 0, len(g.Players))
	dealerScore := GetBestScore(g.Dealer.Cards)
	dealerBJ := IsBlackjack(g.Dealer.Cards)

	// Process all player hands (including split hands)
	for handID, hand := range g.Players {
		// Determine the player ID for this hand
		playerID := handID

		// If this is a split hand, the handID might be different from the playerID
		// In that case, we need to get the actual playerID from the parent hand ID
		if hand.IsSplit() && hand.GetParentHandID() != "" {
			playerID = hand.GetParentHandID()
		}

		// Create base result
		result := HandResult{
			PlayerID: playerID,
			HandID:   handID,
			Score:    hand.Value(),
			Bet:      g.Bets[handID],
		}

		// Check if this is a split hand
		if hand.IsSplit() {
			result.IsSplit = true
			result.ParentHandID = hand.GetParentHandID()
		}

		// Check for double down
		if hand.IsDoubledDown() {
			result.IsDoubledDown = true
			result.DoubleDownBet = hand.GetDoubleDownBet()
			// Adjust the total bet to include the double down amount
			result.Bet += result.DoubleDownBet
		}

		// Check for insurance
		if hand.HasInsurance() {
			result.HasInsurance = true
			result.InsuranceBet = hand.GetInsuranceBet()
		}

		// Handle busted hands first
		if hand.Status == StatusBust {
			result.Result = entities.StringResultLose
			result.Payout = 0 // No payout for busted hands
			results = append(results, result)
			continue
		}

		// Check for player blackjack (only for non-split hands)
		if IsBlackjack(hand.Cards) && !hand.IsSplit() {
			if dealerBJ {
				result.Result = entities.StringResultPush
				result.Payout = result.Bet // Return the original bet
			} else {
				result.Result = entities.StringResultBlackjack
				result.Payout = result.Bet + int64(float64(result.Bet)*1.5) // Blackjack pays 3:2
			}

			// Calculate insurance payout if applicable
			if result.HasInsurance && dealerBJ {
				result.InsurancePayout = result.InsuranceBet * 2 // Insurance pays 2:1
			} else if result.HasInsurance {
				result.InsurancePayout = 0 // No insurance payout if dealer doesn't have blackjack
			}

			results = append(results, result)
			continue
		}

		// Compare scores for non-blackjack hands
		playerScore := result.Score

		if dealerBJ {
			result.Result = entities.StringResultLose
			result.Payout = 0 // No payout for loss

			// Calculate insurance payout if applicable
			if result.HasInsurance {
				result.InsurancePayout = result.InsuranceBet * 2 // Insurance pays 2:1
			}
		} else if hand.Status == StatusBust {
			result.Result = entities.StringResultLose
			result.Payout = 0 // No payout for busted hands
		} else if g.Dealer.Status == StatusBust {
			result.Result = entities.StringResultWin
			result.Payout = result.Bet * 2 // Win pays 1:1 (return original bet + equal amount)
		} else if playerScore > dealerScore {
			result.Result = entities.StringResultWin
			result.Payout = result.Bet * 2 // Win pays 1:1 (return original bet + equal amount)
		} else if playerScore < dealerScore {
			result.Result = entities.StringResultLose
			result.Payout = 0 // No payout for loss
		} else {
			result.Result = entities.StringResultPush
			result.Payout = result.Bet // Return the original bet
		}

		// Calculate insurance payout if applicable and not already calculated
		if result.HasInsurance && dealerBJ && result.InsurancePayout == 0 {
			result.InsurancePayout = result.InsuranceBet * 2 // Insurance pays 2:1
		}

		results = append(results, result)
	}

	return results, nil
}

// AnyPlayerEligibleForSplit checks if any player is eligible for split
func (g *Game) AnyPlayerEligibleForSplit() bool {
	for _, playerID := range g.PlayerOrder {
		if g.IsEligibleForSplit(playerID) {
			return true
		}
	}
	return false
}

// AnyPlayerEligibleForSpecialBets checks if any player is eligible for special bets (double down or insurance)
func (g *Game) AnyPlayerEligibleForSpecialBets() bool {
	for _, playerID := range g.PlayerOrder {
		if g.IsEligibleForDoubleDown(playerID) || g.IsEligibleForInsurance() {
			return true
		}
	}
	return false
}

// BlackjackGameDetails implements the entities.GameDetails interface for blackjack games
type BlackjackGameDetails struct {
	DealerCards []string `json:"dealer_cards"`
	DealerScore int      `json:"dealer_score"`
}

// GameType returns the game type
func (d BlackjackGameDetails) GameType() entities.GameState {
	return entities.GameState("blackjack")
}

// ValidateDetails ensures the details are valid
func (d BlackjackGameDetails) ValidateDetails() error {
	// Basic validation
	if d.DealerScore < 0 || d.DealerScore > 30 {
		return errors.New("invalid dealer score")
	}
	return nil
}

// Helper functions for game record conversion

// getCardStrings converts a slice of Card pointers to a slice of strings
func getCardStrings(cards []*entities.Card) []string {
	result := make([]string, len(cards))
	for i, card := range cards {
		result[i] = card.String()
	}
	return result
}

// convertStringResultToResult converts a StringResult to the Result interface
func convertStringResultToResult(result entities.StringResult) entities.Result {
	return result // StringResult already implements the Result interface
}

// convertToGameResult converts a GameRecord to a GameResult for saving
func convertToGameResult(record game.GameRecord) *entities.GameResult {
	// Create base game result
	result := &entities.GameResult{
		ChannelID:     record.ChannelID,
		GameType:      entities.GameState(record.GameType),
		CompletedAt:   record.EndTime,
		PlayerResults: make([]*entities.PlayerResult, 0, len(record.PlayerRecords)),
		Details: BlackjackGameDetails{
			DealerCards: record.DealerCards,
			DealerScore: record.DealerScore,
		},
	}

	// Convert player records to player results
	for _, handRecord := range record.PlayerRecords {
		// Create player result
		playerResult := &entities.PlayerResult{
			PlayerID: handRecord.PlayerID,
			Result:   handRecord.Result,
			Score:    handRecord.FinalScore,
			Bet:      handRecord.InitialBet,
			Payout:   handRecord.Payout,
			Metadata: make(map[string]interface{}),
		}

		// Add special bet information to metadata
		if handRecord.IsSplit {
			playerResult.Metadata["is_split"] = true
			playerResult.Metadata["hand_id"] = handRecord.HandID
			if handRecord.ParentHandID != "" {
				playerResult.Metadata["parent_hand_id"] = handRecord.ParentHandID
			}
		}
		if handRecord.IsDoubledDown {
			playerResult.Metadata["is_doubled_down"] = true
			playerResult.Metadata["double_down_bet"] = handRecord.DoubleDownBet
		}
		if handRecord.HasInsurance {
			playerResult.Metadata["has_insurance"] = true
			playerResult.Metadata["insurance_bet"] = handRecord.InsuranceBet
			playerResult.Metadata["insurance_payout"] = handRecord.InsurancePayout
		}

		// Add score information
		playerResult.Metadata["score"] = handRecord.FinalScore
		playerResult.Metadata["bet"] = handRecord.InitialBet

		// Add to result
		result.PlayerResults = append(result.PlayerResults, playerResult)
	}

	return result
}
