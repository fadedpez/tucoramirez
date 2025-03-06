package file

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/storage"
	"github.com/stretchr/testify/suite"
)

type StorageTestSuite struct {
	suite.Suite
	tempDir string
	storage *Storage
}

func TestStorage(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}

func (s *StorageTestSuite) SetupTest() {
	// Create temp directory for test files
	tempDir, err := os.MkdirTemp("", "game-storage-test")
	s.Require().NoError(err)
	s.tempDir = tempDir

	// Create storage instance
	options := &storage.Options{
		Path:        filepath.Join(tempDir, "games.db"),
		MaxGameAge:  time.Hour,
		AutoCleanup: false,
	}
	storage, err := New(options)
	s.Require().NoError(err)
	s.storage = storage
}

func (s *StorageTestSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

func (s *StorageTestSuite) TestSaveAndLoadGame() {
	// Setup
	ctx := context.Background()
	game := &storage.GameState{
		ID:        "test-game",
		GameType:  "blackjack",
		ChannelID: "channel-1",
		CreatorID: "player-1",
		State:     []byte(`{"deck":[],"players":{}}`),
	}

	// Execute
	err := s.storage.SaveGame(ctx, game)
	s.Require().NoError(err, "Failed to save game")

	// Assert
	loaded, err := s.storage.LoadGame(ctx, game.ID)
	s.Require().NoError(err, "Failed to load game")
	s.Equal(game.ID, loaded.ID, "Game ID mismatch")
	s.Equal(game.GameType, loaded.GameType, "Game type mismatch")
	s.Equal(game.ChannelID, loaded.ChannelID, "Channel ID mismatch")
	s.Equal(game.CreatorID, loaded.CreatorID, "Creator ID mismatch")
	s.Equal(game.State, loaded.State, "Game state mismatch")
	s.False(loaded.CreatedAt.IsZero(), "Created time not set")
	s.False(loaded.UpdatedAt.IsZero(), "Updated time not set")
}

func (s *StorageTestSuite) TestLoadGameByChannel() {
	// Setup
	ctx := context.Background()
	game := &storage.GameState{
		ID:        "test-game",
		GameType:  "blackjack",
		ChannelID: "channel-1",
		CreatorID: "player-1",
		State:     []byte(`{"deck":[],"players":{}}`),
	}
	s.Require().NoError(s.storage.SaveGame(ctx, game))

	// Execute
	loaded, err := s.storage.LoadGameByChannel(ctx, game.ChannelID)

	// Assert
	s.Require().NoError(err, "Failed to load game by channel")
	s.Equal(game.ID, loaded.ID, "Game ID mismatch")
}

func (s *StorageTestSuite) TestDeleteGame() {
	// Setup
	ctx := context.Background()
	game := &storage.GameState{
		ID:        "test-game",
		GameType:  "blackjack",
		ChannelID: "channel-1",
		CreatorID: "player-1",
	}
	s.Require().NoError(s.storage.SaveGame(ctx, game))

	// Execute
	err := s.storage.DeleteGame(ctx, game.ID)

	// Assert
	s.Require().NoError(err, "Failed to delete game")
	_, err = s.storage.LoadGame(ctx, game.ID)
	s.Error(err, "Game should be deleted")
}

func (s *StorageTestSuite) TestListGames() {
	// Setup
	ctx := context.Background()
	games := []*storage.GameState{
		{
			ID:        "game-1",
			GameType:  "blackjack",
			ChannelID: "channel-1",
			CreatorID: "player-1",
		},
		{
			ID:        "game-2",
			GameType:  "blackjack",
			ChannelID: "channel-2",
			CreatorID: "player-2",
		},
	}
	for _, game := range games {
		s.Require().NoError(s.storage.SaveGame(ctx, game))
	}

	// Execute
	listed, err := s.storage.ListGames(ctx)

	// Assert
	s.Require().NoError(err, "Failed to list games")
	s.Len(listed, len(games), "Wrong number of games")
}

func (s *StorageTestSuite) TestCleanupOldGames() {
	// Setup
	ctx := context.Background()
	oldGame := &storage.GameState{
		ID:        "old-game",
		GameType:  "blackjack",
		ChannelID: "channel-1",
		CreatorID: "player-1",
		UpdatedAt: time.Now().Add(-2 * time.Hour),
	}
	newGame := &storage.GameState{
		ID:        "new-game",
		GameType:  "blackjack",
		ChannelID: "channel-2",
		CreatorID: "player-2",
		UpdatedAt: time.Now(),
	}
	s.Require().NoError(s.storage.SaveGame(ctx, oldGame))
	s.Require().NoError(s.storage.SaveGame(ctx, newGame))

	// Execute
	err := s.storage.CleanupOldGames(ctx, time.Hour)

	// Assert
	s.Require().NoError(err, "Failed to cleanup old games")
	games, err := s.storage.ListGames(ctx)
	s.Require().NoError(err)
	s.Len(games, 1, "Should have only one game after cleanup")
	s.Equal(newGame.ID, games[0].ID, "Wrong game remained after cleanup")
}
