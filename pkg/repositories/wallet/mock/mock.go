// Code generated by MockGen. DO NOT EDIT.
// Source: interface.go
//
// Generated by this command:
//
//	mockgen -source=interface.go -destination=mock/mock.go -package=mock_wallet
//
// Package mock_wallet is a generated GoMock package.
package mock_wallet

import (
	context "context"
	reflect "reflect"

	entities "github.com/fadedpez/tucoramirez/pkg/entities"
	gomock "go.uber.org/mock/gomock"
)

// MockRepository is a mock of Repository interface.
type MockRepository struct {
	ctrl     *gomock.Controller
	recorder *MockRepositoryMockRecorder
}

// MockRepositoryMockRecorder is the mock recorder for MockRepository.
type MockRepositoryMockRecorder struct {
	mock *MockRepository
}

// NewMockRepository creates a new mock instance.
func NewMockRepository(ctrl *gomock.Controller) *MockRepository {
	mock := &MockRepository{ctrl: ctrl}
	mock.recorder = &MockRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockRepository) EXPECT() *MockRepositoryMockRecorder {
	return m.recorder
}

// AddTransaction mocks base method.
func (m *MockRepository) AddTransaction(ctx context.Context, transaction *entities.Transaction) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddTransaction", ctx, transaction)
	ret0, _ := ret[0].(error)
	return ret0
}

// AddTransaction indicates an expected call of AddTransaction.
func (mr *MockRepositoryMockRecorder) AddTransaction(ctx, transaction any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddTransaction", reflect.TypeOf((*MockRepository)(nil).AddTransaction), ctx, transaction)
}

// GetTransactions mocks base method.
func (m *MockRepository) GetTransactions(ctx context.Context, userID string, limit int) ([]*entities.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransactions", ctx, userID, limit)
	ret0, _ := ret[0].([]*entities.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTransactions indicates an expected call of GetTransactions.
func (mr *MockRepositoryMockRecorder) GetTransactions(ctx, userID, limit any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransactions", reflect.TypeOf((*MockRepository)(nil).GetTransactions), ctx, userID, limit)
}

// GetTransactionsByType mocks base method.
func (m *MockRepository) GetTransactionsByType(ctx context.Context, userID string, transactionType entities.TransactionType, limit int) ([]*entities.Transaction, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransactionsByType", ctx, userID, transactionType, limit)
	ret0, _ := ret[0].([]*entities.Transaction)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTransactionsByType indicates an expected call of GetTransactionsByType.
func (mr *MockRepositoryMockRecorder) GetTransactionsByType(ctx, userID, transactionType, limit any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransactionsByType", reflect.TypeOf((*MockRepository)(nil).GetTransactionsByType), ctx, userID, transactionType, limit)
}

// GetWallet mocks base method.
func (m *MockRepository) GetWallet(ctx context.Context, userID string) (*entities.Wallet, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetWallet", ctx, userID)
	ret0, _ := ret[0].(*entities.Wallet)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetWallet indicates an expected call of GetWallet.
func (mr *MockRepositoryMockRecorder) GetWallet(ctx, userID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetWallet", reflect.TypeOf((*MockRepository)(nil).GetWallet), ctx, userID)
}

// SaveWallet mocks base method.
func (m *MockRepository) SaveWallet(ctx context.Context, wallet *entities.Wallet) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveWallet", ctx, wallet)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveWallet indicates an expected call of SaveWallet.
func (mr *MockRepositoryMockRecorder) SaveWallet(ctx, wallet any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveWallet", reflect.TypeOf((*MockRepository)(nil).SaveWallet), ctx, wallet)
}

// UpdateBalance mocks base method.
func (m *MockRepository) UpdateBalance(ctx context.Context, userID string, amount int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBalance", ctx, userID, amount)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateBalance indicates an expected call of UpdateBalance.
func (mr *MockRepositoryMockRecorder) UpdateBalance(ctx, userID, amount any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBalance", reflect.TypeOf((*MockRepository)(nil).UpdateBalance), ctx, userID, amount)
}
