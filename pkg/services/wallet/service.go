package wallet

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
	walletRepo "github.com/fadedpez/tucoramirez/pkg/repositories/wallet"
	"github.com/google/uuid"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrNegativeAmount    = errors.New("amount cannot be negative")
)

// Service handles wallet business logic
type Service struct {
	repo walletRepo.Repository
}

// NewService creates a new wallet service
func NewService(repo walletRepo.Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// GetOrCreateWallet retrieves a wallet or creates a new one if it doesn't exist
func (s *Service) GetOrCreateWallet(ctx context.Context, userID string) (*entities.Wallet, bool, error) {
	wallet, err := s.repo.GetWallet(ctx, userID)
	if err == nil {
		return wallet, false, nil // Wallet exists
	}

	if !errors.Is(err, walletRepo.ErrWalletNotFound) {
		return nil, false, err // Unexpected error
	}

	// Create a new wallet with starting balance of 100
	newWallet := &entities.Wallet{
		UserID:      userID,
		Balance:     100, // Starting balance
		LoanAmount:  0,
		LastUpdated: time.Now(),
	}

	if err := s.repo.SaveWallet(ctx, newWallet); err != nil {
		return nil, false, err
	}

	return newWallet, true, nil
}

// GetBalance returns the current balance for a user
func (s *Service) GetBalance(ctx context.Context, userID string) (int64, error) {
	wallet, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		return 0, err
	}
	return wallet.Balance, nil
}

// AddFunds adds funds to a user's wallet
func (s *Service) AddFunds(ctx context.Context, userID string, amount int64, description string) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}

	// Log the start of the operation
	log.Printf("[WALLET] Adding $%d to wallet for user %s with description: %s", amount, userID, description)

	wallet, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		log.Printf("[WALLET] Error getting wallet for user %s: %v", userID, err)
		return err
	}

	// Log the wallet state before update
	log.Printf("[WALLET] Before update - User %s: Balance=$%d, LoanAmount=$%d", userID, wallet.Balance, wallet.LoanAmount)

	// Update the wallet balance directly
	wallet.Balance += amount
	wallet.LastUpdated = time.Now()
	
	// Save the updated wallet
	if err := s.repo.SaveWallet(ctx, wallet); err != nil {
		log.Printf("[WALLET] Error saving wallet for user %s: %v", userID, err)
		return err
	}

	// Log the wallet state after update
	log.Printf("[WALLET] After update - User %s: Balance=$%d, LoanAmount=$%d", userID, wallet.Balance, wallet.LoanAmount)

	// Record the transaction
	transaction := &entities.Transaction{
		ID:           uuid.New().String(),
		UserID:       userID,
		Amount:       amount,
		Type:         entities.TransactionTypeLoan, // Default type, should be overridden by caller
		Description:  description,
		Timestamp:    time.Now(),
		BalanceAfter: wallet.Balance,
	}

	// Log the transaction
	log.Printf("[WALLET] Recording transaction: ID=%s, User=%s, Amount=$%d, Type=%s", 
		transaction.ID, userID, amount, transaction.Type)

	err = s.repo.AddTransaction(ctx, transaction)
	if err != nil {
		log.Printf("[WALLET] Error adding transaction for user %s: %v", userID, err)
	}
	return err
}

// RemoveFunds removes funds from a user's wallet if sufficient funds exist
func (s *Service) RemoveFunds(ctx context.Context, userID string, amount int64, description string) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}

	wallet, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		return err
	}

	// Check if user has sufficient funds
	if wallet.Balance < amount {
		return ErrInsufficientFunds
	}

	// Update the wallet balance directly
	wallet.Balance -= amount
	wallet.LastUpdated = time.Now()
	
	// Save the updated wallet
	if err := s.repo.SaveWallet(ctx, wallet); err != nil {
		return err
	}

	// Record the transaction
	transaction := &entities.Transaction{
		ID:           uuid.New().String(),
		UserID:       userID,
		Amount:       -amount, // Negative amount for removal
		Type:         entities.TransactionTypeRepayment, // Default type, should be overridden by caller
		Description:  description,
		Timestamp:    time.Now(),
		BalanceAfter: wallet.Balance,
	}

	return s.repo.AddTransaction(ctx, transaction)
}

// TakeLoan adds a loan amount to the user's wallet
func (s *Service) TakeLoan(ctx context.Context, userID string, amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}

	wallet, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		return err
	}

	// Update the wallet balance directly in the wallet object
	wallet.Balance += amount
	
	// Update the loan amount
	wallet.LoanAmount += amount
	wallet.LastUpdated = time.Now()
	
	// Save the updated wallet with both balance and loan changes
	if err := s.repo.SaveWallet(ctx, wallet); err != nil {
		return err
	}

	// Record the transaction
	transaction := &entities.Transaction{
		ID:           uuid.New().String(),
		UserID:       userID,
		Amount:       amount,
		Type:         entities.TransactionTypeLoan,
		Description:  "Loan from Tuco",
		Timestamp:    time.Now(),
		BalanceAfter: wallet.Balance,
	}

	return s.repo.AddTransaction(ctx, transaction)
}

// RepayLoan repays a portion of the user's loan
func (s *Service) RepayLoan(ctx context.Context, userID string, amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}

	wallet, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		return err
	}

	// Check if user has sufficient funds
	if wallet.Balance < amount {
		return ErrInsufficientFunds
	}

	// Check if repayment exceeds loan amount
	if amount > wallet.LoanAmount {
		amount = wallet.LoanAmount // Cap at loan amount
	}

	// Update the wallet balance directly in the wallet object
	wallet.Balance -= amount
	
	// Update the loan amount
	wallet.LoanAmount -= amount
	wallet.LastUpdated = time.Now()
	
	// Save the updated wallet with both balance and loan changes
	if err := s.repo.SaveWallet(ctx, wallet); err != nil {
		return err
	}

	// Record the transaction
	transaction := &entities.Transaction{
		ID:           uuid.New().String(),
		UserID:       userID,
		Amount:       -amount, // Negative amount for removal
		Type:         entities.TransactionTypeRepayment,
		Description:  "Loan repayment to Tuco",
		Timestamp:    time.Now(),
		BalanceAfter: wallet.Balance,
	}

	return s.repo.AddTransaction(ctx, transaction)
}

// GetRecentTransactions retrieves recent transactions for a user
func (s *Service) GetRecentTransactions(ctx context.Context, userID string, limit int) ([]*entities.Transaction, error) {
	return s.repo.GetTransactions(ctx, userID, limit)
}
