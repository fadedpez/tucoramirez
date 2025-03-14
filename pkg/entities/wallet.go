package entities

import (
	"time"
)

// Wallet represents a player's currency inventory
type Wallet struct {
	UserID      string    // Discord user ID
	Balance     int64     // Current balance in dollars
	LoanAmount  int64     // Amount currently loaned to the player
	LastUpdated time.Time // When the wallet was last updated
}

// TransactionType represents the type of wallet transaction
type TransactionType string

const (
	TransactionTypeBet       TransactionType = "BET"
	TransactionTypePayout    TransactionType = "PAYOUT"
	TransactionTypeLoan      TransactionType = "LOAN"
	TransactionTypeRepayment TransactionType = "REPAYMENT"
)

// Transaction represents a single wallet transaction
type Transaction struct {
	ID           string          // Unique identifier
	UserID       string          // User associated with the transaction
	Amount       int64           // Amount (positive for additions, negative for subtractions)
	Type         TransactionType // Type of transaction
	ReferenceID  string          // Optional reference (e.g., game ID for bets)
	Description  string          // Human-readable description
	Timestamp    time.Time       // When the transaction occurred
	BalanceAfter int64           // Balance after this transaction
}
