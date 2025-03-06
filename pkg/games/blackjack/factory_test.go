package blackjack

import (
	"testing"

	discordmock "github.com/fadedpez/tucoramirez/internal/discord/mock"
	"github.com/fadedpez/tucoramirez/internal/games"
	"github.com/fadedpez/tucoramirez/pkg/storage"
	"github.com/stretchr/testify/suite"
)

type FactorySuite struct {
	suite.Suite
	factory *Factory
	session *discordmock.SessionHandler
	storage *storage.MockStorage
}

func (s *FactorySuite) SetupTest() {
	s.session = &discordmock.SessionHandler{}
	s.session.Test(s.T())
	s.storage = storage.NewMockStorage(s.T())
	s.factory = NewFactory(s.session, s.storage)
}

func TestFactorySuite(t *testing.T) {
	suite.Run(t, new(FactorySuite))
}

func (s *FactorySuite) TestNewFactory() {
	s.NotNil(s.factory)
	s.Equal(s.session, s.factory.session)
	s.Equal(s.storage, s.factory.storage)
}

func (s *FactorySuite) TestCreateGame() {
	// Test creating game with no players
	game := s.factory.CreateGame("creator123", "channel456", []string{})
	s.NotNil(game)
	s.IsType(&Game{}, game)

	// Test creating game with players
	players := []string{"player1", "player2"}
	game = s.factory.CreateGame("creator123", "channel456", players)
	s.NotNil(game)
	blackjackGame := game.(*Game)
	s.Len(blackjackGame.Players, 2)
	s.Contains(blackjackGame.Players, "player1")
	s.Contains(blackjackGame.Players, "player2")
}

func (s *FactorySuite) TestCreateManager() {
	manager := s.factory.CreateManager()
	s.NotNil(manager)
	s.Implements((*games.Manager)(nil), manager)

	// Verify it's a blackjack manager
	blackjackManager, ok := manager.(*Manager)
	s.True(ok)
	s.Equal(s.storage, blackjackManager.storage)
}
