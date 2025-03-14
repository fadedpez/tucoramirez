package wallet

import (
	"context"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

//go:generate mockgen -source=$GOFILE -destination=mock/mock.go -package=mock_wallet_service
type WalletService interface {
	GetOrCreateWallet(ctx context.Context, userID string) (*entities.Wallet, bool, error)
	AddFunds(ctx context.Context, userID string, amount int64, description string) error
}
