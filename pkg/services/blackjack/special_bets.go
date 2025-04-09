package blackjack

import (
	"context"
	"errors"
	"log"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

// IsEligibleForDoubleDown checks if a player is eligible for double down
func (g *Game) IsEligibleForDoubleDown(playerID string) bool {
	// Check if player has exactly 2 cards and hasn't already acted
	hand, exists := g.Players[playerID]
	if !exists || len(hand.Cards) != 2 || hand.IsDoubledDown() {
		return false
	}

	// Can't double down if already split
	if hand.IsSplit() {
		return false
	}

	return true
}

// IsEligibleForSplit checks if a player is eligible for split
func (g *Game) IsEligibleForSplit(playerID string) bool {
	// Check if player has exactly 2 cards of the same rank
	hand, exists := g.Players[playerID]
	if !exists || len(hand.Cards) != 2 {
		return false
	}

	// Can't split if already doubled down or split
	if hand.IsDoubledDown() || hand.IsSplit() {
		return false
	}

	// Check if the two cards have the same rank
	return hand.Cards[0].Rank == hand.Cards[1].Rank
}

// IsEligibleForInsurance checks if a player is eligible for insurance
func (g *Game) IsEligibleForInsurance() bool {
	// Check if dealer's up card is an Ace
	if len(g.Dealer.Cards) < 1 {
		return false
	}

	return g.Dealer.Cards[0].Rank == entities.Ace
}

// DoubleDown performs a double down action for a player
func (g *Game) DoubleDown(ctx context.Context, playerID string, walletService WalletService) error {
	// Validate game state
	if g.State != StateSpecialBets {
		return ErrInvalidAction
	}

	// Validate player turn
	currentPlayerID, err := g.GetCurrentSpecialBetsPlayerID()
	if err != nil {
		return err
	}
	if currentPlayerID != playerID {
		return ErrNotPlayerTurn
	}

	// Check if player is eligible for double down
	if !g.IsEligibleForDoubleDown(playerID) {
		return ErrNotEligibleForDoubleDown
	}

	// Get the player's hand and bet amount
	hand, exists := g.Players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	betAmount, exists := g.Bets[playerID]
	if !exists {
		return errors.New("no bet found for player")
	}

	// Ensure player has enough funds for the double down bet
	_, _, err = walletService.EnsureFundsWithLoan(ctx, playerID, betAmount, walletService.GetStandardLoanIncrement())
	if err != nil {
		return err
	}

	// Remove funds for the double down bet
	err = walletService.RemoveFunds(ctx, playerID, betAmount, "Double down bet")
	if err != nil {
		return err
	}

	// Mark the hand as doubled down and store the bet amount
	hand.SetDoubledDown(true)
	hand.SetDoubleDownBet(betAmount)

	// Deal one more card to the player
	newCard := g.Deck.Draw()
	if newCard == nil {
		// Reshuffle if deck is empty
		g.Deck = entities.NewDeck()
		g.Deck.Shuffle()
		g.shuffled = true
		newCard = g.Deck.Draw()
	}
	err = hand.AddCard(newCard)
	if err != nil {
		// If there's an error adding the card, it's likely not a bust error
		// since we're adding just one card to a valid hand
		return err
	}

	// After double down, the hand is automatically marked as standing
	// unless the player busted
	if hand.Status != StatusBust {
		err = hand.Stand()
		if err != nil {
			return err
		}
	}

	// Advance to the next player's turn
	return g.AdvanceSpecialBetsTurn()
}

// Split performs a split action for a player
func (g *Game) Split(ctx context.Context, playerID string, walletService WalletService) error {
	// Validate game state
	if g.State != StateSplitting {
		return ErrInvalidAction
	}

	// Validate player turn
	if g.GetCurrentSplittingPlayerID() != playerID {
		return ErrNotPlayerTurn
	}

	// Check if player is eligible for split
	if !g.IsEligibleForSplit(playerID) {
		return ErrNotEligibleForSplit
	}

	// Get the player's hand and bet amount
	hand, exists := g.Players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	betAmount, exists := g.Bets[playerID]
	if !exists {
		return errors.New("no bet found for player")
	}

	// Ensure player has enough funds for the split bet
	_, _, err := walletService.EnsureFundsWithLoan(ctx, playerID, betAmount, walletService.GetStandardLoanIncrement())
	if err != nil {
		return err
	}

	// Remove funds for the split bet
	err = walletService.RemoveFunds(ctx, playerID, betAmount, "Split bet")
	if err != nil {
		return err
	}

	// Create a new hand for the split
	splitHandID := playerID + "_split"
	splitHand := NewHand()

	// Mark both hands as split
	hand.SetSplit(true)
	splitHand.SetSplit(true)

	// Set parent-child relationship
	splitHand.SetParentHandID(playerID)

	// Move the second card to the split hand
	secondCard := hand.Cards[1]
	hand.Cards = hand.Cards[:1]
	splitHand.AddCard(secondCard)

	// Deal one more card to each hand
	newCardForOriginal := g.Deck.Draw()
	if newCardForOriginal == nil {
		// Reshuffle if deck is empty
		g.Deck = entities.NewDeck()
		g.Deck.Shuffle()
		g.shuffled = true
		newCardForOriginal = g.Deck.Draw()
	}
	hand.AddCard(newCardForOriginal)

	newCardForSplit := g.Deck.Draw()
	if newCardForSplit == nil {
		// Reshuffle if deck is empty
		g.Deck = entities.NewDeck()
		g.Deck.Shuffle()
		g.shuffled = true
		newCardForSplit = g.Deck.Draw()
	}
	splitHand.AddCard(newCardForSplit)

	// Add the split hand to the game
	g.Players[splitHandID] = splitHand

	// Add the bet for the split hand
	g.Bets[splitHandID] = betAmount

	// Update the player order to include the split hand right after the original hand
	// This ensures the player plays both hands in sequence
	newPlayerOrder := make([]string, 0, len(g.PlayerOrder)+1)
	for _, id := range g.PlayerOrder {
		newPlayerOrder = append(newPlayerOrder, id)
		// Insert the split hand right after the original hand
		if id == playerID {
			newPlayerOrder = append(newPlayerOrder, splitHandID)
		}
	}
	g.PlayerOrder = newPlayerOrder

	// Advance to the next player's turn
	return g.AdvanceSplittingTurn()
}

// PlaceInsurance performs an insurance bet for a player
func (g *Game) PlaceInsurance(ctx context.Context, playerID string, walletService WalletService) error {
	// Validate game state
	if g.State != StateSpecialBets {
		return ErrInvalidAction
	}

	// Validate player turn
	currentPlayerID, err := g.GetCurrentSpecialBetsPlayerID()
	if err != nil {
		return err
	}
	if currentPlayerID != playerID {
		return ErrNotPlayerTurn
	}

	// Check if player is eligible for insurance
	if !g.IsEligibleForInsurance() {
		return ErrNotEligibleForInsurance
	}

	// Get the player's hand and bet amount
	hand, exists := g.Players[playerID]
	if !exists {
		return ErrPlayerNotFound
	}

	betAmount, exists := g.Bets[playerID]
	if !exists {
		return errors.New("no bet found for player")
	}

	// Insurance bet is half the original bet
	insuranceAmount := betAmount / 2

	// Ensure player has enough funds for the insurance bet
	_, _, err = walletService.EnsureFundsWithLoan(ctx, playerID, insuranceAmount, walletService.GetStandardLoanIncrement())
	if err != nil {
		return err
	}

	// Remove funds for the insurance bet
	err = walletService.RemoveFunds(ctx, playerID, insuranceAmount, "Insurance bet")
	if err != nil {
		return err
	}

	// Mark the hand as having insurance and store the bet amount
	hand.SetInsurance(true)
	hand.SetInsuranceBet(insuranceAmount)

	// Advance to the next player's turn
	return g.AdvanceSpecialBetsTurn()
}

// DeclineSpecialBet allows a player to decline any special betting options
func (g *Game) DeclineSpecialBet(playerID string) error {
	// Validate game state
	if g.State != StateSpecialBets {
		return ErrInvalidAction
	}

	// Validate player turn
	currentPlayerID, err := g.GetCurrentSpecialBetsPlayerID()
	if err != nil {
		return err
	}
	if currentPlayerID != playerID {
		return ErrNotPlayerTurn
	}

	// Simply advance to the next player's turn
	return g.AdvanceSpecialBetsTurn()
}

// AdvanceSpecialBetsTurn advances to the next player's turn for special bets
func (g *Game) AdvanceSpecialBetsTurn() error {
	// Validate game state
	if g.State != StateSpecialBets {
		return ErrInvalidAction
	}

	// Get the current player IDs in order
	playerIDs := g.getPlayerIDsInOrder()

	// Increment the current turn
	g.CurrentSpecialBetsTurn++

	// Check if we've gone through all players
	if g.CurrentSpecialBetsTurn >= len(playerIDs) {
		// All players have had a chance to make special bets
		// Always transition to playing phase after special bets
		log.Printf("All players have had a chance at special bets, transitioning to PLAYING state")
		g.State = entities.StatePlaying
		g.CurrentTurn = 0
		
		// Skip players who are already done (bust or stand)
		// This ensures players who doubled down and automatically stood are skipped
		for g.CheckPlayerDone(g.PlayerOrder[g.CurrentTurn]) && !g.CheckAllPlayersDone() {
			log.Printf("Skipping player %s who is already done", g.PlayerOrder[g.CurrentTurn])
			g.CurrentTurn = (g.CurrentTurn + 1) % len(g.PlayerOrder)
		}
		
		// If all players are done after skipping, transition to dealer phase
		if g.CheckAllPlayersDone() {
			log.Printf("All players are done after special bets, transitioning to dealer phase")
			g.State = entities.StateDealer
		}
		
		return nil
	}

	// Check if the current player is eligible for any special bets
	currentPlayerID := playerIDs[g.CurrentSpecialBetsTurn]

	// If the player isn't eligible for any special bets, skip to the next player
	if !g.IsEligibleForDoubleDown(currentPlayerID) &&
		!g.IsEligibleForSplit(currentPlayerID) &&
		!g.IsEligibleForInsurance() {
		return g.AdvanceSpecialBetsTurn()
	}

	return nil
}

// GetCurrentSpecialBetsPlayerID returns the ID of the player whose turn it is to make special bets
func (g *Game) GetCurrentSpecialBetsPlayerID() (string, error) {
	if g.State != StateSpecialBets {
		return "", ErrInvalidAction
	}

	playerIDs := g.getPlayerIDsInOrder()

	if g.CurrentSpecialBetsTurn >= len(playerIDs) {
		return "", errors.New("no current player for special bets")
	}

	return playerIDs[g.CurrentSpecialBetsTurn], nil
}

// getPlayerIDsInOrder returns the player IDs in the order they should take their turns
func (g *Game) getPlayerIDsInOrder() []string {
	// If we have a predefined player order, use that
	if len(g.PlayerOrder) > 0 {
		return g.PlayerOrder
	}

	// Otherwise, get the player IDs from the Players map
	playerIDs := make([]string, 0, len(g.Players))
	for playerID := range g.Players {
		// Skip split hands when determining turn order
		hand := g.Players[playerID]
		if hand.IsSplit() && hand.GetParentHandID() != "" {
			continue
		}
		playerIDs = append(playerIDs, playerID)
	}

	return playerIDs
}

// GetCurrentSplittingPlayerID returns the ID of the player whose turn it is to split
func (g *Game) GetCurrentSplittingPlayerID() string {
	if g.State != StateSplitting || g.CurrentSpecialBetsTurn >= len(g.PlayerOrder) {
		return ""
	}

	return g.PlayerOrder[g.CurrentSpecialBetsTurn]
}

// AdvanceSplittingTurn advances to the next player's turn during the splitting phase
func (g *Game) AdvanceSplittingTurn() error {
	// Validate game state
	if g.State != StateSplitting {
		return ErrInvalidAction
	}

	// Increment the current turn
	g.CurrentSpecialBetsTurn++

	// Check if we've gone through all players
	if g.CurrentSpecialBetsTurn >= len(g.PlayerOrder) {
		// All players have had a chance to split, move to special bets or playing state
		if g.AnyPlayerEligibleForSpecialBets() {
			g.State = StateSpecialBets
			g.CurrentSpecialBetsTurn = 0
		} else {
			g.State = entities.StatePlaying
			g.CurrentTurn = 0
		}
		return nil
	}

	// Check if the current player is eligible for split
	currentPlayerID := g.PlayerOrder[g.CurrentSpecialBetsTurn]

	// If the player isn't eligible for split, skip to the next player
	if !g.IsEligibleForSplit(currentPlayerID) {
		return g.AdvanceSplittingTurn()
	}

	return nil
}

// DeclineSplit allows a player to skip splitting
func (g *Game) DeclineSplit(playerID string) error {
	// Validate game state
	if g.State != StateSplitting {
		return ErrInvalidAction
	}

	// Validate player turn
	if g.GetCurrentSplittingPlayerID() != playerID {
		return ErrNotPlayerTurn
	}

	// Simply advance to the next player's turn
	return g.AdvanceSplittingTurn()
}
