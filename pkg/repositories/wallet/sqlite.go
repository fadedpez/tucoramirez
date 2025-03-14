package wallet

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

// SQLite table schemas
const (
	createWalletsTableSQL = `
	CREATE TABLE IF NOT EXISTS wallets (
		user_id TEXT PRIMARY KEY,
		balance INTEGER NOT NULL DEFAULT 100,
		loan_amount INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	createTransactionsTableSQL = `
	CREATE TABLE IF NOT EXISTS transactions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		amount INTEGER NOT NULL,
		type TEXT NOT NULL,
		reference_id TEXT,
		description TEXT,
		timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		balance_after INTEGER NOT NULL,
		FOREIGN KEY (user_id) REFERENCES wallets(user_id)
	)`

	createTransactionIndexesSQL = `
	CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
	CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
	CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(timestamp DESC)
	`
)

// SQLiteRepository implements Repository using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	// Ensure directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating database directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Create tables if they don't exist
	if _, err := db.Exec(createWalletsTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("error creating wallets table: %w", err)
	}

	if _, err := db.Exec(createTransactionsTableSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("error creating transactions table: %w", err)
	}

	if _, err := db.Exec(createTransactionIndexesSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("error creating transaction indexes: %w", err)
	}

	return &SQLiteRepository{db: db}, nil
}

// GetWallet retrieves a wallet by user ID
func (r *SQLiteRepository) GetWallet(ctx context.Context, userID string) (*entities.Wallet, error) {
	query := `SELECT user_id, balance, loan_amount, updated_at FROM wallets WHERE user_id = ?`

	var wallet entities.Wallet
	var updatedAt string

	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&wallet.UserID,
		&wallet.Balance,
		&wallet.LoanAmount,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("error getting wallet: %w", err)
	}

	// Parse the timestamp
	// Try parsing with different formats since SQLite might store timestamps in different formats
	formats := []string{
		"2006-01-02 15:04:05",  // SQLite default format
		"2006-01-02T15:04:05Z",  // ISO 8601 format
		"2006-01-02T15:04:05-07:00", // ISO 8601 with timezone
		time.RFC3339,              // Another common format
	}

	var parseErr error
	for _, format := range formats {
		wallet.LastUpdated, parseErr = time.Parse(format, updatedAt)
		if parseErr == nil {
			break
		}
	}

	if parseErr != nil {
		return nil, fmt.Errorf("error parsing timestamp '%s': %w", updatedAt, parseErr)
	}

	return &wallet, nil
}

// SaveWallet creates or updates a wallet
func (r *SQLiteRepository) SaveWallet(ctx context.Context, wallet *entities.Wallet) error {
	// Use a standardized timestamp format (SQLite default format)
	formattedTime := wallet.LastUpdated.Format("2006-01-02 15:04:05")
	
	// Log the wallet save operation
	log.Printf("[WALLET_REPO] Saving wallet for user %s: Balance=$%d, LoanAmount=$%d, Time=%s", 
		wallet.UserID, wallet.Balance, wallet.LoanAmount, formattedTime)
	
	query := `
		INSERT INTO wallets (user_id, balance, loan_amount, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			balance = ?,
			loan_amount = ?,
			updated_at = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		wallet.UserID, wallet.Balance, wallet.LoanAmount, formattedTime,
		wallet.Balance, wallet.LoanAmount, formattedTime,
	)

	if err != nil {
		log.Printf("[WALLET_REPO] Error saving wallet for user %s: %v", wallet.UserID, err)
		return fmt.Errorf("error saving wallet: %w", err)
	}

	// Verify the wallet was saved correctly by retrieving it again
	updatedWallet, err := r.GetWallet(ctx, wallet.UserID)
	if err != nil {
		log.Printf("[WALLET_REPO] Error verifying wallet save for user %s: %v", wallet.UserID, err)
		return fmt.Errorf("error verifying wallet save: %w", err)
	}

	// Log the verification result
	log.Printf("[WALLET_REPO] Verified wallet save for user %s: Balance=$%d, LoanAmount=$%d", 
		updatedWallet.UserID, updatedWallet.Balance, updatedWallet.LoanAmount)

	return nil
}

// UpdateBalance atomically updates a wallet's balance
func (r *SQLiteRepository) UpdateBalance(ctx context.Context, userID string, amount int64) error {
	// Use a standardized timestamp format (SQLite default format)
	formattedTime := time.Now().Format("2006-01-02 15:04:05")

	query := `
		UPDATE wallets
		SET balance = balance + ?,
			updated_at = ?
		WHERE user_id = ?
	`

	result, err := r.db.ExecContext(ctx, query, amount, formattedTime, userID)
	if err != nil {
		return fmt.Errorf("error updating balance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrWalletNotFound
	}

	return nil
}

// AddTransaction records a new transaction
func (r *SQLiteRepository) AddTransaction(ctx context.Context, transaction *entities.Transaction) error {
	// Generate ID if not provided
	if transaction.ID == "" {
		transaction.ID = uuid.New().String()
	}

	// Set timestamp if not provided
	if transaction.Timestamp.IsZero() {
		transaction.Timestamp = time.Now()
	}

	query := `
		INSERT INTO transactions (
			id, user_id, amount, type, reference_id, description, timestamp, balance_after
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		transaction.ID,
		transaction.UserID,
		transaction.Amount,
		transaction.Type,
		transaction.ReferenceID,
		transaction.Description,
		transaction.Timestamp.Format("2006-01-02 15:04:05"),
		transaction.BalanceAfter,
	)

	if err != nil {
		return fmt.Errorf("error adding transaction: %w", err)
	}

	return nil
}

// GetTransactions retrieves recent transactions for a user
func (r *SQLiteRepository) GetTransactions(ctx context.Context, userID string, limit int) ([]*entities.Transaction, error) {
	query := `
		SELECT id, user_id, amount, type, reference_id, description, timestamp, balance_after
		FROM transactions
		WHERE user_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("error querying transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.Transaction

	for rows.Next() {
		var tx entities.Transaction
		var timestamp string

		err := rows.Scan(
			&tx.ID,
			&tx.UserID,
			&tx.Amount,
			&tx.Type,
			&tx.ReferenceID,
			&tx.Description,
			&timestamp,
			&tx.BalanceAfter,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning transaction row: %w", err)
		}

		// Parse the timestamp
		// Try parsing with different formats since SQLite might store timestamps in different formats
		formats := []string{
			"2006-01-02 15:04:05",  // SQLite default format
			"2006-01-02T15:04:05Z",  // ISO 8601 format
			"2006-01-02T15:04:05-07:00", // ISO 8601 with timezone
			time.RFC3339,              // Another common format
		}

		var parseErr error
		for _, format := range formats {
			tx.Timestamp, parseErr = time.Parse(format, timestamp)
			if parseErr == nil {
				break
			}
		}

		if parseErr != nil {
			return nil, fmt.Errorf("error parsing timestamp '%s': %w", timestamp, parseErr)
		}

		transactions = append(transactions, &tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows: %w", err)
	}

	return transactions, nil
}

// GetTransactionsByType retrieves transactions of a specific type
func (r *SQLiteRepository) GetTransactionsByType(ctx context.Context, userID string, transactionType entities.TransactionType, limit int) ([]*entities.Transaction, error) {
	query := `
		SELECT id, user_id, amount, type, reference_id, description, timestamp, balance_after
		FROM transactions
		WHERE user_id = ? AND type = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, userID, transactionType, limit)
	if err != nil {
		return nil, fmt.Errorf("error querying transactions by type: %w", err)
	}
	defer rows.Close()

	var transactions []*entities.Transaction

	for rows.Next() {
		var tx entities.Transaction
		var timestamp string

		err := rows.Scan(
			&tx.ID,
			&tx.UserID,
			&tx.Amount,
			&tx.Type,
			&tx.ReferenceID,
			&tx.Description,
			&timestamp,
			&tx.BalanceAfter,
		)

		if err != nil {
			return nil, fmt.Errorf("error scanning transaction row: %w", err)
		}

		// Parse the timestamp
		// Try parsing with different formats since SQLite might store timestamps in different formats
		formats := []string{
			"2006-01-02 15:04:05",  // SQLite default format
			"2006-01-02T15:04:05Z",  // ISO 8601 format
			"2006-01-02T15:04:05-07:00", // ISO 8601 with timezone
			time.RFC3339,              // Another common format
		}

		var parseErr error
		for _, format := range formats {
			tx.Timestamp, parseErr = time.Parse(format, timestamp)
			if parseErr == nil {
				break
			}
		}

		if parseErr != nil {
			return nil, fmt.Errorf("error parsing timestamp '%s': %w", timestamp, parseErr)
		}

		transactions = append(transactions, &tx)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transaction rows: %w", err)
	}

	return transactions, nil
}

// Close closes the database connection
func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}
