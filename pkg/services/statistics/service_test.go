package statistics

import (
	"context"
	"testing"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the game.Repository interface
type MockRepository struct {
	mock.Mock
}

// GetAllPlayerStatistics is a mock implementation of the Repository.GetAllPlayerStatistics method
func (m *MockRepository) GetAllPlayerStatistics(ctx context.Context, gameType entities.GameState) ([]*entities.PlayerStatistics, error) {
	args := m.Called(ctx, gameType)
	return args.Get(0).([]*entities.PlayerStatistics), args.Error(1)
}

// UpdatePlayerStatistics is a mock implementation of the Repository.UpdatePlayerStatistics method
func (m *MockRepository) UpdatePlayerStatistics(ctx context.Context, gameResult *entities.GameResult) error {
	args := m.Called(ctx, gameResult)
	return args.Error(0)
}

// SaveDeck implements Repository
func (m *MockRepository) SaveDeck(ctx context.Context, channelID string, deck []*entities.Card) error {
	return nil
}

// GetDeck implements Repository
func (m *MockRepository) GetDeck(ctx context.Context, channelID string) ([]*entities.Card, error) {
	return nil, nil
}

// SaveGameResult implements Repository
func (m *MockRepository) SaveGameResult(ctx context.Context, result *entities.GameResult) error {
	return nil
}

// GetPlayerResults implements Repository
func (m *MockRepository) GetPlayerResults(ctx context.Context, playerID string) ([]*entities.GameResult, error) {
	return nil, nil
}

// GetChannelResults implements Repository
func (m *MockRepository) GetChannelResults(ctx context.Context, channelID string, limit int) ([]*entities.GameResult, error) {
	return nil, nil
}

// GetPlayerStatistics implements Repository
func (m *MockRepository) GetPlayerStatistics(ctx context.Context, playerID string, gameType entities.GameState) (*entities.PlayerStatistics, error) {
	return nil, nil
}

// PruneGameResultsPerPlayer implements Repository
func (m *MockRepository) PruneGameResultsPerPlayer(ctx context.Context, maxMatchesPerPlayer int) error {
	return nil
}

// ArchiveGameResults implements Repository
func (m *MockRepository) ArchiveGameResults(ctx context.Context, gameResults []*entities.GameResult) error {
	return nil
}

// IndexGameResult implements Repository
func (m *MockRepository) IndexGameResult(ctx context.Context, gameResult *entities.GameResult) error {
	return nil
}

// GetPlayerStatisticsFromES implements Repository
func (m *MockRepository) GetPlayerStatisticsFromES(ctx context.Context, playerID string, gameType entities.GameState) (*entities.PlayerStatistics, error) {
	return nil, nil
}

// GetAllPlayerStatisticsFromES implements Repository
func (m *MockRepository) GetAllPlayerStatisticsFromES(ctx context.Context, gameType entities.GameState) ([]*entities.PlayerStatistics, error) {
	return nil, nil
}

// Close implements Repository
func (m *MockRepository) Close() error {
	return nil
}

// TestGetBlackjackLeaderboard tests the GetBlackjackLeaderboard method
func TestGetBlackjackLeaderboard(t *testing.T) {
	// Create a mock repository
	mockRepo := new(MockRepository)

	// Create test data
	testStats := []*entities.PlayerStatistics{
		{
			PlayerID:      "player1",
			GameType:      "blackjack",
			GamesPlayed:   10,
			Wins:          5,
			Losses:        4,
			Pushes:        1,
			Blackjacks:    2,
			Busts:         3,
			Splits:        1,
			DoubleDowns:   2,
			TotalBet:      1000,
			TotalWinnings: 1500,
			LastUpdated:   time.Now(),
		},
		{
			PlayerID:      "player2",
			GameType:      "blackjack",
			GamesPlayed:   15,
			Wins:          8,
			Losses:        5,
			Pushes:        2,
			Blackjacks:    3,
			Busts:         4,
			Splits:        2,
			DoubleDowns:   3,
			TotalBet:      2000,
			TotalWinnings: 2800,
			LastUpdated:   time.Now(),
		},
		{
			PlayerID:      "player3",
			GameType:      "blackjack",
			GamesPlayed:   20,
			Wins:          12,
			Losses:        6,
			Pushes:        2,
			Blackjacks:    4,
			Busts:         5,
			Splits:        3,
			DoubleDowns:   4,
			TotalBet:      3000,
			TotalWinnings: 4000,
			LastUpdated:   time.Now(),
		},
	}

	// Set up the mock repository to return the test data
	mockRepo.On("GetAllPlayerStatistics", mock.Anything, entities.GameState("blackjack")).Return(testStats, nil)

	// Create the service with the mock repository
	service := NewService(mockRepo)

	// Call the method being tested
	ctx := context.Background()
	leaderboard, err := service.GetBlackjackLeaderboard(ctx, 1, 10)

	// Assert that there was no error
	assert.NoError(t, err)

	// Assert that the leaderboard was created correctly
	assert.NotNil(t, leaderboard)
	assert.Equal(t, 3, leaderboard.TotalPlayers)
	assert.Equal(t, 1, leaderboard.CurrentPage)
	assert.Equal(t, 1, leaderboard.TotalPages)
	assert.Equal(t, 10, leaderboard.PlayersPerPage)

	// Assert that the players are sorted by total winnings (descending)
	assert.Equal(t, 3, len(leaderboard.Players))
	assert.Equal(t, "player3", leaderboard.Players[0].PlayerID)
	assert.Equal(t, "player2", leaderboard.Players[1].PlayerID)
	assert.Equal(t, "player1", leaderboard.Players[2].PlayerID)

	// Assert that the ranks are assigned correctly
	assert.Equal(t, 1, leaderboard.Players[0].Rank)
	assert.Equal(t, 2, leaderboard.Players[1].Rank)
	assert.Equal(t, 3, leaderboard.Players[2].Rank)

	// Assert that the top winner and top player flags are set correctly
	assert.True(t, leaderboard.Players[0].IsTopWinner)
	assert.True(t, leaderboard.Players[0].IsTopPlayer) // player3 has the most games played

	// Verify that the mock was called as expected
	mockRepo.AssertExpectations(t)
}

// TestEnsureStatisticsUpdated tests the EnsureStatisticsUpdated method
func TestEnsureStatisticsUpdated(t *testing.T) {
	// Create a mock repository
	mockRepo := new(MockRepository)

	// Create a test game result
	gameResult := &entities.GameResult{
		ChannelID:   "channel1",
		GameType:    "blackjack",
		CompletedAt: time.Now(),
		PlayerResults: []*entities.PlayerResult{
			{
				PlayerID: "player1",
				Result:   entities.StringResultWin,
				Score:    21,
				Bet:      100,
				Payout:   200,
			},
			{
				PlayerID: "player2",
				Result:   entities.StringResultLose,
				Score:    18,
				Bet:      100,
				Payout:   0,
			},
		},
	}

	// Set up the mock repository to expect UpdatePlayerStatistics to be called
	mockRepo.On("UpdatePlayerStatistics", mock.Anything, gameResult).Return(nil)

	// Create the service with the mock repository
	service := NewService(mockRepo)

	// Call the method being tested
	ctx := context.Background()
	err := service.EnsureStatisticsUpdated(ctx, gameResult)

	// Assert that there was no error
	assert.NoError(t, err)

	// Verify that the mock was called as expected
	mockRepo.AssertExpectations(t)
}
