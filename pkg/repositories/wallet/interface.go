package wallet

import (
	"context"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

// Repository defines the interface for wallet data operations
type Repository interface {
	// GetWallet retrieves a wallet by user ID
	GetWallet(ctx context.Context, userID string) (*entities.Wallet, error)

	// SaveWallet creates or updates a wallet
	SaveWallet(ctx context.Context, wallet *entities.Wallet) error

	// UpdateBalance atomically updates a wallet's balance
	UpdateBalance(ctx context.Context, userID string, amount int64) error

	// AddTransaction records a new transaction
	AddTransaction(ctx context.Context, transaction *entities.Transaction) error

	// GetTransactions retrieves recent transactions for a user
	GetTransactions(ctx context.Context, userID string, limit int) ([]*entities.Transaction, error)

	// GetTransactionsByType retrieves transactions of a specific type
	GetTransactionsByType(ctx context.Context, userID string, transactionType entities.TransactionType, limit int) ([]*entities.Transaction, error)
}
