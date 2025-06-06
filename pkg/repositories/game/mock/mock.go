// Code generated by MockGen. DO NOT EDIT.
// Source: interface.go
//
// Generated by this command:
//
//	mockgen -source=interface.go -destination=mock/mock.go -package=mock_game
//
// Package mock_game is a generated GoMock package.
package mock_game

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

// Close mocks base method.
func (m *MockRepository) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close.
func (mr *MockRepositoryMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockRepository)(nil).Close))
}

// GetChannelResults mocks base method.
func (m *MockRepository) GetChannelResults(ctx context.Context, channelID string, limit int) ([]*entities.GameResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetChannelResults", ctx, channelID, limit)
	ret0, _ := ret[0].([]*entities.GameResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetChannelResults indicates an expected call of GetChannelResults.
func (mr *MockRepositoryMockRecorder) GetChannelResults(ctx, channelID, limit any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetChannelResults", reflect.TypeOf((*MockRepository)(nil).GetChannelResults), ctx, channelID, limit)
}

// GetDeck mocks base method.
func (m *MockRepository) GetDeck(ctx context.Context, channelID string) ([]*entities.Card, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDeck", ctx, channelID)
	ret0, _ := ret[0].([]*entities.Card)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDeck indicates an expected call of GetDeck.
func (mr *MockRepositoryMockRecorder) GetDeck(ctx, channelID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDeck", reflect.TypeOf((*MockRepository)(nil).GetDeck), ctx, channelID)
}

// GetPlayerResults mocks base method.
func (m *MockRepository) GetPlayerResults(ctx context.Context, playerID string) ([]*entities.GameResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPlayerResults", ctx, playerID)
	ret0, _ := ret[0].([]*entities.GameResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPlayerResults indicates an expected call of GetPlayerResults.
func (mr *MockRepositoryMockRecorder) GetPlayerResults(ctx, playerID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPlayerResults", reflect.TypeOf((*MockRepository)(nil).GetPlayerResults), ctx, playerID)
}

// SaveDeck mocks base method.
func (m *MockRepository) SaveDeck(ctx context.Context, channelID string, deck []*entities.Card) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveDeck", ctx, channelID, deck)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveDeck indicates an expected call of SaveDeck.
func (mr *MockRepositoryMockRecorder) SaveDeck(ctx, channelID, deck any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveDeck", reflect.TypeOf((*MockRepository)(nil).SaveDeck), ctx, channelID, deck)
}

// SaveGameResult mocks base method.
func (m *MockRepository) SaveGameResult(ctx context.Context, result *entities.GameResult) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveGameResult", ctx, result)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveGameResult indicates an expected call of SaveGameResult.
func (mr *MockRepositoryMockRecorder) SaveGameResult(ctx, result any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveGameResult", reflect.TypeOf((*MockRepository)(nil).SaveGameResult), ctx, result)
}
