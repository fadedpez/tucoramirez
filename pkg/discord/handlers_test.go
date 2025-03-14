package discord

import (
	"context"

	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/fadedpez/tucoramirez/pkg/services/blackjack"
	"github.com/fadedpez/tucoramirez/pkg/services/wallet"
	"github.com/stretchr/testify/mock"
)

// MockWalletService is a mock implementation of the wallet service
type MockWalletService struct {
	mock.Mock
	*wallet.Service
}

func (m *MockWalletService) GetOrCreateWallet(ctx context.Context, userID string) (*entities.Wallet, bool, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*entities.Wallet), args.Bool(1), args.Error(2)
}

func (m *MockWalletService) AddFunds(ctx context.Context, userID string, amount int64, description string) error {
	args := m.Called(ctx, userID, amount, description)
	return args.Error(0)
}

func (m *MockWalletService) RemoveFunds(ctx context.Context, userID string, amount int64, description string) error {
	args := m.Called(ctx, userID, amount, description)
	return args.Error(0)
}

// MockGame is a mock implementation of the blackjack game for testing
type MockGame struct {
	blackjack.Game
}

// ProcessPayouts mocks the ProcessPayouts method of the blackjack game
func (m *MockGame) ProcessPayouts() map[string]int64 {
	// Return a predefined map of payouts for testing
	return map[string]int64{
		"player1": 200, // Win: bet * 2 = 200
		"player2": 500, // Blackjack: bet + (bet * 3 / 2) = 500
		"player3": 150, // Push: original bet = 150
		"player4": 0,   // Loss: no payout = 0
	}
}
