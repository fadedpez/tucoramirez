package wallet

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/google/uuid"
)

var (
	ErrWalletNotFound = errors.New("wallet not found")
)

// MemoryRepository implements Repository using in-memory storage
type MemoryRepository struct {
	wallets      map[string]*entities.Wallet
	transactions map[string][]*entities.Transaction
	mu           sync.RWMutex
}

// NewMemoryRepository creates a new in-memory wallet repository
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		wallets:      make(map[string]*entities.Wallet),
		transactions: make(map[string][]*entities.Transaction),
	}
}

// GetWallet retrieves a wallet by user ID
func (r *MemoryRepository) GetWallet(ctx context.Context, userID string) (*entities.Wallet, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	wallet, exists := r.wallets[userID]
	if !exists {
		return nil, ErrWalletNotFound
	}

	// Return a copy to prevent concurrent modification
	walletCopy := *wallet
	return &walletCopy, nil
}

// SaveWallet creates or updates a wallet
func (r *MemoryRepository) SaveWallet(ctx context.Context, wallet *entities.Wallet) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Update the last updated timestamp
	wallet.LastUpdated = time.Now()

	// Create a copy to prevent concurrent modification
	walletCopy := *wallet
	r.wallets[wallet.UserID] = &walletCopy

	return nil
}

// UpdateBalance atomically updates a wallet's balance
func (r *MemoryRepository) UpdateBalance(ctx context.Context, userID string, amount int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	wallet, exists := r.wallets[userID]
	if !exists {
		return ErrWalletNotFound
	}

	wallet.Balance += amount
	wallet.LastUpdated = time.Now()

	return nil
}

// AddTransaction records a new transaction
func (r *MemoryRepository) AddTransaction(ctx context.Context, transaction *entities.Transaction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate a UUID if not provided
	if transaction.ID == "" {
		transaction.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if transaction.Timestamp.IsZero() {
		transaction.Timestamp = time.Now()
	}

	// Make a copy to prevent concurrent modification
	txCopy := *transaction

	// Initialize the transactions slice if it doesn't exist
	if _, exists := r.transactions[transaction.UserID]; !exists {
		r.transactions[transaction.UserID] = make([]*entities.Transaction, 0)
	}

	r.transactions[transaction.UserID] = append(r.transactions[transaction.UserID], &txCopy)

	return nil
}

// GetTransactions retrieves recent transactions for a user
func (r *MemoryRepository) GetTransactions(ctx context.Context, userID string, limit int) ([]*entities.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	transactions, exists := r.transactions[userID]
	if !exists {
		return make([]*entities.Transaction, 0), nil
	}

	// Create a copy of the transactions
	result := make([]*entities.Transaction, 0, limit)

	// Get the most recent transactions up to the limit
	start := 0
	if len(transactions) > limit {
		start = len(transactions) - limit
	}

	for i := start; i < len(transactions); i++ {
		txCopy := *transactions[i]
		result = append(result, &txCopy)
	}

	return result, nil
}

// GetTransactionsByType retrieves transactions of a specific type
func (r *MemoryRepository) GetTransactionsByType(ctx context.Context, userID string, transactionType entities.TransactionType, limit int) ([]*entities.Transaction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	transactions, exists := r.transactions[userID]
	if !exists {
		return make([]*entities.Transaction, 0), nil
	}

	// Filter transactions by type
	filtered := make([]*entities.Transaction, 0, limit)
	for i := len(transactions) - 1; i >= 0 && len(filtered) < limit; i-- {
		if transactions[i].Type == transactionType {
			txCopy := *transactions[i]
			filtered = append(filtered, &txCopy)
		}
	}

	return filtered, nil
}
