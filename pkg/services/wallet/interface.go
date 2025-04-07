package wallet

import (
	"context"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

//go:generate mockgen -source=$GOFILE -destination=mock/mock.go -package=mock_wallet_service
type WalletService interface {
	GetOrCreateWallet(ctx context.Context, userID string) (*entities.Wallet, bool, error)
	AddFunds(ctx context.Context, userID string, amount int64, description string) error
	RemoveFunds(ctx context.Context, userID string, amount int64, description string) error
	EnsureFundsWithLoan(ctx context.Context, userID string, requiredAmount int64, loanAmount int64) (*entities.Wallet, bool, error)
	ValidateLoan(ctx context.Context, userID string, amount int64) error
	GiveLoan(ctx context.Context, userID string, amount int64) (*entities.Wallet, bool, error)
	ValidateRepayment(ctx context.Context, userID string, amount int64) error
	RepayLoan(ctx context.Context, userID string, amount int64) error
	GetStandardLoanIncrement() int64
	CalculateRepaymentAmount(ctx context.Context, userID string) (int64, error)
	CanRepayLoan(ctx context.Context, userID string) (bool, error)
}