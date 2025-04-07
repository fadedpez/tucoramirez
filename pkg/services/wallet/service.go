package wallet

import (
	"context"
	"errors"
	"fmt"
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
	// Validate the repayment first
	err := s.ValidateRepayment(ctx, userID, amount)
	if err != nil {
		return err
	}

	wallet, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		return err
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

// EnsureFundsWithLoan checks if a user has enough funds for a specified amount.
// If not, it automatically gives them a loan of the specified loan amount.
// Returns the updated wallet, whether a loan was given, and any error.
func (s *Service) EnsureFundsWithLoan(ctx context.Context, userID string, requiredAmount int64, loanAmount int64) (*entities.Wallet, bool, error) {
	// Get the current wallet
	wallet, _, err := s.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return nil, false, fmt.Errorf("error getting wallet: %w", err)
	}

	// Check if they have enough funds
	if wallet.Balance >= requiredAmount {
		// They have enough funds, no loan needed
		return wallet, false, nil
	}

	// They need a loan
	err = s.TakeLoan(ctx, userID, loanAmount)
	if err != nil {
		return nil, false, fmt.Errorf("error taking loan: %w", err)
	}

	// Get the updated wallet after the loan
	updatedWallet, _, err := s.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return nil, false, fmt.Errorf("error getting updated wallet: %w", err)
	}

	// Return the updated wallet and indicate a loan was given
	return updatedWallet, true, nil
}

// ValidateRepayment checks if a user can repay their loan
func (s *Service) ValidateRepayment(ctx context.Context, userID string, amount int64) error {
	wallet, _, err := s.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return fmt.Errorf("error getting wallet: %w", err)
	}
	
	if wallet.LoanAmount <= 0 {
		return errors.New("no loan to repay")
	}
	
	// Ensure repayments are in increments of 100
	if amount % 100 != 0 {
		return errors.New("repayment amount must be in increments of 100")
	}
	
	if wallet.Balance < amount {
		return errors.New("insufficient funds to repay loan")
	}
	
	// Check if repayment exceeds loan amount
	if amount > wallet.LoanAmount {
		return fmt.Errorf("repayment amount exceeds loan amount of %d", wallet.LoanAmount)
	}
	
	return nil
}

// ValidateLoan checks if a loan amount is valid
func (s *Service) ValidateLoan(ctx context.Context, userID string, amount int64) error {
	if amount <= 0 {
		return ErrNegativeAmount
	}
	
	// Ensure loans are in increments of 100
	if amount % 100 != 0 {
		return errors.New("loan amount must be in increments of 100")
	}
	
	return nil
}

// GiveLoan gives a loan to a user
func (s *Service) GiveLoan(ctx context.Context, userID string, amount int64) (*entities.Wallet, bool, error) {
	// Validate the loan first
	err := s.ValidateLoan(ctx, userID, amount)
	if err != nil {
		return nil, false, err
	}

	// Get the user's wallet
	wallet, _, err := s.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return nil, false, fmt.Errorf("error getting wallet: %w", err)
	}

	// Add the loan amount to the wallet
	wallet.Balance += amount
	wallet.LoanAmount += amount // Add to existing loan amount instead of replacing
	wallet.LastUpdated = time.Now()

	// Save the updated wallet
	if err := s.repo.SaveWallet(ctx, wallet); err != nil {
		return nil, false, fmt.Errorf("error updating wallet: %w", err)
	}

	// Record the transaction
	transaction := &entities.Transaction{
		ID:          uuid.New().String(),
		UserID:      userID,
		Amount:      amount,
		Type:        entities.TransactionTypeLoan,
		Description: "Loan from Tuco",
		Timestamp:   time.Now(),
		BalanceAfter: wallet.Balance,
	}

	err = s.repo.AddTransaction(ctx, transaction)
	if err != nil {
		log.Printf("Failed to record loan transaction: %v", err)
	}

	return wallet, true, nil
}

// GetStandardLoanIncrement returns the standard increment for loans
func (s *Service) GetStandardLoanIncrement() int64 {
	return 100 // Loans are in increments of 100
}

// CalculateRepaymentAmount calculates the appropriate repayment amount
func (s *Service) CalculateRepaymentAmount(ctx context.Context, userID string) (int64, error) {
	wallet, _, err := s.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("error getting wallet: %w", err)
	}
	
	if wallet.LoanAmount <= 0 {
		return 0, errors.New("no loan to repay")
	}
	
	// Return the standard loan increment or the full loan amount if it's smaller
	repaymentAmount := s.GetStandardLoanIncrement()
	if wallet.LoanAmount < repaymentAmount {
		repaymentAmount = wallet.LoanAmount
	}
	
	return repaymentAmount, nil
}

// CanRepayLoan checks if a user has a loan that can be repaid
func (s *Service) CanRepayLoan(ctx context.Context, userID string) (bool, error) {
	wallet, _, err := s.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("error getting wallet: %w", err)
	}
	
	return wallet.LoanAmount > 0, nil
}
