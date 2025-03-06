package games

import (
	"testing"

	discordmock "github.com/fadedpez/tucoramirez/internal/discord/mock"
	"github.com/fadedpez/tucoramirez/internal/types"
	"github.com/stretchr/testify/suite"
)

type RegistryTestSuite struct {
	suite.Suite
	registry *Registry
	factory  *MockFactory
	session  *discordmock.SessionHandler
}

func TestRegistrySuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}

func (s *RegistryTestSuite) SetupTest() {
	s.registry = NewRegistry()
	s.factory = &MockFactory{}
	s.session = &discordmock.SessionHandler{}
	s.session.Test(s.T())
}

func (s *RegistryTestSuite) TestNewRegistry() {
	// Execute
	registry := NewRegistry()

	// Assert
	s.NotNil(registry, "Registry should not be nil")
	s.NotNil(registry.factories, "Factories map should be initialized")
	s.Empty(registry.factories, "Factories map should be empty")
}

func (s *RegistryTestSuite) TestRegisterGame() {
	// Setup
	gameName := "test_game"

	// Execute
	err := s.registry.RegisterGame(gameName, s.factory)

	// Assert
	s.NoError(err, "Should register game without error")
	
	// Verify game is registered
	factory, exists := s.registry.factories[gameName]
	s.True(exists, "Game should be registered")
	s.Equal(s.factory, factory, "Registered factory should match")
}

func (s *RegistryTestSuite) TestRegisterGameDuplicate() {
	// Setup
	gameName := "test_game"
	s.registry.RegisterGame(gameName, s.factory)

	// Execute
	err := s.registry.RegisterGame(gameName, s.factory)

	// Assert
	s.Error(err, "Should return error when registering duplicate game")
	s.True(types.IsGameError(err, types.ErrInvalidAction), "Should return InvalidAction error")
}

func (s *RegistryTestSuite) TestGetFactory() {
	// Setup
	gameName := "test_game"
	s.registry.RegisterGame(gameName, s.factory)

	// Execute
	factory, err := s.registry.GetFactory(gameName)

	// Assert
	s.NoError(err, "Should get factory without error")
	s.Equal(s.factory, factory, "Should return correct factory")
}

func (s *RegistryTestSuite) TestGetFactoryNotFound() {
	// Execute
	factory, err := s.registry.GetFactory("nonexistent_game")

	// Assert
	s.Error(err, "Should return error for nonexistent game")
	s.Nil(factory, "Factory should be nil")
	s.True(types.IsGameError(err, types.ErrGameNotFound), "Should return GameNotFound error")
}

func (s *RegistryTestSuite) TestCreateManager() {
	// Setup
	gameName := "test_game"
	mockManager := &MockManager{}
	s.factory.On("CreateManager").Return(mockManager)
	s.registry.RegisterGame(gameName, s.factory)

	// Execute
	manager, err := s.registry.CreateManager(gameName, s.session)

	// Assert
	s.NoError(err, "Should create manager without error")
	s.Equal(mockManager, manager, "Should return correct manager")
	s.factory.AssertExpectations(s.T())
}

func (s *RegistryTestSuite) TestCreateManagerGameNotFound() {
	// Execute
	manager, err := s.registry.CreateManager("nonexistent_game", s.session)

	// Assert
	s.Error(err, "Should return error for nonexistent game")
	s.Nil(manager, "Manager should be nil")
	s.True(types.IsGameError(err, types.ErrGameNotFound), "Should return GameNotFound error")
}

func (s *RegistryTestSuite) TestListGames() {
	// Setup
	games := []string{"game1", "game2", "game3"}
	for _, game := range games {
		s.registry.RegisterGame(game, s.factory)
	}

	// Execute
	registeredGames := s.registry.ListGames()

	// Assert
	s.ElementsMatch(games, registeredGames, "Should return all registered games")
}

func (s *RegistryTestSuite) TestConcurrentAccess() {
	// Setup
	done := make(chan bool)
	numGoroutines := 10
	numOperations := 100

	// Execute concurrent reads and writes
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				// Mix of read and write operations
				if j%2 == 0 {
					gameName := "game_write_" + string(rune(id))
					s.registry.RegisterGame(gameName, s.factory)
				} else {
					s.registry.ListGames()
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Assert no data race occurred (test will fail if race detector finds issues)
	s.NotEmpty(s.registry.ListGames(), "Registry should contain games after concurrent operations")
}
