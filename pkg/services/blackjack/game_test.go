package blackjack

import (
	"context"
	"testing"

	"github.com/fadedpez/tucoramirez/pkg/entities"
	mock_game "github.com/fadedpez/tucoramirez/pkg/repositories/game/mock"
	mock_wallet "github.com/fadedpez/tucoramirez/pkg/repositories/wallet/mock"
	mock_wallet_service "github.com/fadedpez/tucoramirez/pkg/services/wallet/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"strings"
)

// GameTestSuite is a test suite for the Game service
type GameTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	mockGameRepo   *mock_game.MockRepository
	mockWalletRepo *mock_wallet.MockRepository
	game           *Game
	channelID      string
	testDeck       *entities.Deck
}

// SetupTest sets up the test suite
func (s *GameTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockGameRepo = mock_game.NewMockRepository(s.ctrl)
	s.mockWalletRepo = mock_wallet.NewMockRepository(s.ctrl)

	s.channelID = "test-channel"
	s.game = NewGame(s.channelID, s.mockGameRepo)

	s.testDeck = NewBlackjackDeck()
	s.testDeck.Shuffle()
}

// TearDownTest tears down the test suite
func (s *GameTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

// TestGameSuite runs the test suite
func TestGameSuite(t *testing.T) {
	suite.Run(t, new(GameTestSuite))
}

// TestAddPlayer tests adding a player to the game
func (s *GameTestSuite) TestAddPlayer_GameInProgress() {
	// Add a player
	s.game.State = entities.StateBetting

	err := s.game.AddPlayer("player1")
	s.EqualError(err, ErrGameInProgress.Error())
}

// TestAddPlayer tests adding a player to the game
func (s *GameTestSuite) TestAddPlayer() {
	// Add a player
	err := s.game.AddPlayer("player1")
	s.NoError(err)
	s.Contains(s.game.Players, "player1")

	s.Equal(1, len(s.game.Players))
}

func (s *GameTestSuite) TestStartGame_NoPlayers() {
	s.game.State = entities.StateBetting

	err := s.game.Start()
	s.EqualError(err, "no players in game")
}

func (s *GameTestSuite) TestStartGame_GameInProgress() {
	s.game.State = entities.StateDealing

	err := s.game.Start()
	s.EqualError(err, ErrGameInProgress.Error())
}

func (s *GameTestSuite) TestStartGame_TransitionsToBetting() {
	// Add players
	_ = s.game.AddPlayer("player1")
	_ = s.game.AddPlayer("player2")

	s.game.State = entities.StateWaiting

	err := s.game.Start()
	s.NoError(err)
	s.Equal(entities.StateBetting, s.game.State)
}

func (s *GameTestSuite) TestStartGame_ChecksAllBetsPlaced() {
	// Add players
	_ = s.game.AddPlayer("player1")
	_ = s.game.AddPlayer("player2")

	s.game.State = entities.StateBetting

	err := s.game.Start()
	s.EqualError(err, "not all players have placed bets")

	s.game.Bets["player1"] = 100

	err = s.game.Start()
	s.EqualError(err, "not all players have placed bets")
}

// TestStartGame tests starting a game
func (s *GameTestSuite) TestStartGame() {
	// Add players
	_ = s.game.AddPlayer("player1")
	_ = s.game.AddPlayer("player2")

	s.Equal(2, len(s.game.Players))
	s.Equal(s.game.State, entities.StateWaiting)

	// Mock the repository calls for Start
	s.mockGameRepo.EXPECT().
		GetDeck(gomock.Any(), "test-channel").
		Return(s.testDeck.Cards, nil)

	_ = s.game.Start()

	cardLength := len(s.testDeck.Cards)

	s.Equal(entities.StateBetting, s.game.State)
	_ = s.game.PlaceBet("player1", 100)

	// we are ready to start the game after this which will save the dec
	s.mockGameRepo.EXPECT().
		SaveDeck(gomock.Any(), "test-channel", gomock.Any()).
		Return(nil)

	_ = s.game.PlaceBet("player2", 200)

	_ = s.game.Start()

	s.Equal(entities.StatePlaying, s.game.State)
	s.Equal(cardLength-6, len(s.game.Deck.Cards))
}

// TestGetResults tests getting game results
func (s *GameTestSuite) TestGetResults() {
	// Setup game with controlled cards for predictable results
	_ = s.game.AddPlayer("player1")

	// Create a specific deck for testing
	s.game.Deck = &entities.Deck{
		Cards: []*entities.Card{
			{Rank: entities.Ten, Suit: entities.Hearts},     // Player card 1
			{Rank: entities.Ten, Suit: entities.Clubs},      // Dealer card 1
			{Rank: entities.Seven, Suit: entities.Spades},   // Player card 2
			{Rank: entities.Seven, Suit: entities.Diamonds}, // Dealer card 2
		},
	}

	// Deal cards manually to ensure predictable hands
	s.game.Players["player1"] = NewHand()
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[0]) // 10
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[2]) // 7

	s.game.Dealer = NewHand()
	s.game.Dealer.AddCard(s.game.Deck.Cards[1]) // 10
	s.game.Dealer.AddCard(s.game.Deck.Cards[3]) // 7

	s.game.State = entities.StatePlaying
	s.game.PlayerOrder = []string{"player1"}

	// Make all players stand
	s.game.Players["player1"].Status = StatusStand
	s.game.Dealer.Status = StatusStand
	s.game.State = entities.StateComplete

	// Mock the SaveGameResult and SaveDeck calls for GetResults
	s.mockGameRepo.EXPECT().
		SaveGameResult(gomock.Any(), gomock.Any()).
		Return(nil)

	s.mockGameRepo.EXPECT().
		SaveDeck(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	// Get results
	results, err := s.game.GetResults()
	s.NoError(err)
	s.Equal(1, len(results))

	// Player should push with dealer (both have 17)
	s.Equal("player1", results[0].PlayerID)
	s.Equal(ResultPush, results[0].Result)
	s.Equal(17, results[0].Score)
}

// TestPayoutCalculation tests the payout calculation for different scenarios
func (s *GameTestSuite) TestPayoutCalculation() {
	// Setup game with controlled cards for predictable results
	_ = s.game.AddPlayer("player1")

	// Create a specific deck for testing - setup for a 21 (not blackjack)
	s.game.Deck = &entities.Deck{
		Cards: []*entities.Card{
			{Rank: entities.Five, Suit: entities.Hearts},   // Player card 1
			{Rank: entities.Ten, Suit: entities.Clubs},     // Dealer card 1
			{Rank: entities.Six, Suit: entities.Spades},    // Player card 2
			{Rank: entities.Five, Suit: entities.Diamonds}, // Dealer card 2
			{Rank: entities.Ten, Suit: entities.Hearts},    // Player hit card (21 total)
		},
	}

	// Deal cards manually to ensure predictable hands
	s.game.Players["player1"] = NewHand()
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[0]) // 5
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[2]) // 6
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[4]) // 10 (21 total, not blackjack)

	s.game.Dealer = NewHand()
	s.game.Dealer.AddCard(s.game.Deck.Cards[1]) // 10
	s.game.Dealer.AddCard(s.game.Deck.Cards[3]) // 5 (15 total)

	s.game.State = entities.StatePlaying
	s.game.PlayerOrder = []string{"player1"}
	s.game.Bets["player1"] = 100

	// Make all players stand
	s.game.Players["player1"].Status = StatusStand
	s.game.Dealer.Status = StatusStand
	s.game.State = entities.StateComplete

	// Capture the game result being saved
	var capturedGameResults []*entities.GameResult
	// ProcessPayouts calls GetResults which calls SaveGameResult again
	// so we need to expect at least two calls
	s.mockGameRepo.EXPECT().
		SaveGameResult(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, result *entities.GameResult) error {
			capturedGameResults = append(capturedGameResults, result)
			return nil
		}).
		MinTimes(1)

	// SaveDeck is also called multiple times
	s.mockGameRepo.EXPECT().
		SaveDeck(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		MinTimes(1)

	// Get results
	results, err := s.game.GetResults()
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal("player1", results[0].PlayerID)
	s.Equal(ResultWin, results[0].Result)
	s.Equal(21, results[0].Score)

	// Create a mock wallet service to verify wallet updates
	mockWalletCtrl := gomock.NewController(s.T())
	defer mockWalletCtrl.Finish()
	mockWalletService := mock_wallet_service.NewMockWalletService(mockWalletCtrl)

	// For regular win, player gets 1:1 payout (bet + equal amount = 200 for a 100 bet)
	mockWalletService.EXPECT().
		GetOrCreateWallet(gomock.Any(), "player1").
		Return(&entities.Wallet{UserID: "player1", Balance: 1000}, false, nil).
		Times(2) // Called before and after adding funds

	// Expect AddFunds to be called with the regular win payout amount
	mockWalletService.EXPECT().
		AddFunds(gomock.Any(), "player1", int64(200), gomock.Any()).
		Return(nil).
		Times(1)

	// Process payouts with wallet updates
	err = s.game.ProcessPayouts(context.Background(), wrapMockWalletService(mockWalletService))
	s.NoError(err)

	// Verify at least one game result was saved
	s.NotEmpty(capturedGameResults, "At least one game result should have been saved")

	// Verify all saved game results have the correct data
	for _, capturedResult := range capturedGameResults {
		s.Equal(s.game.ChannelID, capturedResult.ChannelID)
		s.Equal(entities.StateDealing, capturedResult.GameType)
		s.Equal(1, len(capturedResult.PlayerResults))

		// Verify player result details
		playerResult := capturedResult.PlayerResults[0]
		s.Equal("player1", playerResult.PlayerID)
		s.Equal(entities.Result(ResultWin), playerResult.Result)
		s.Equal(21, playerResult.Score)

		// Verify dealer details
		blackjackDetails, ok := capturedResult.Details.(*BlackjackDetails)
		s.True(ok, "Details should be of type BlackjackDetails")
		s.Equal(15, blackjackDetails.DealerScore)
		s.False(blackjackDetails.IsBlackjack)
		s.False(blackjackDetails.IsBust)
	}
}

// TestPayoutCalculation_DealerWins tests the payout calculation when dealer wins
func (s *GameTestSuite) TestPayoutCalculation_DealerWins() {
	// Setup game with controlled cards for predictable results
	_ = s.game.AddPlayer("player1")

	// Create a specific deck for testing - setup for dealer winning
	s.game.Deck = &entities.Deck{
		Cards: []*entities.Card{
			{Rank: entities.Five, Suit: entities.Hearts},   // Player card 1
			{Rank: entities.Ten, Suit: entities.Clubs},     // Dealer card 1
			{Rank: entities.Six, Suit: entities.Spades},    // Player card 2
			{Rank: entities.Nine, Suit: entities.Diamonds}, // Dealer card 2
			// No additional cards needed for this scenario
		},
	}

	// Deal cards manually to ensure predictable hands
	s.game.Players["player1"] = NewHand()
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[0]) // 5
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[2]) // 6 (11 total)

	s.game.Dealer = NewHand()
	s.game.Dealer.AddCard(s.game.Deck.Cards[1]) // 10
	s.game.Dealer.AddCard(s.game.Deck.Cards[3]) // 9 (19 total)

	s.game.State = entities.StatePlaying
	s.game.PlayerOrder = []string{"player1"}
	s.game.Bets["player1"] = 100

	// Make all players stand
	s.game.Players["player1"].Status = StatusStand
	s.game.Dealer.Status = StatusStand
	s.game.State = entities.StateComplete

	// Capture the game result being saved
	var capturedGameResults []*entities.GameResult
	// ProcessPayouts calls GetResults which calls SaveGameResult again
	// so we need to expect at least two calls
	s.mockGameRepo.EXPECT().
		SaveGameResult(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, result *entities.GameResult) error {
			capturedGameResults = append(capturedGameResults, result)
			return nil
		}).
		MinTimes(1)

	// SaveDeck is also called multiple times
	s.mockGameRepo.EXPECT().
		SaveDeck(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		MinTimes(1)

	// Get results - player1 should lose to dealer (11 vs 19)
	results, err := s.game.GetResults()
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal("player1", results[0].PlayerID)
	s.Equal(ResultLose, results[0].Result)
	s.Equal(11, results[0].Score)

	// Create a mock wallet service to verify wallet updates
	mockWalletCtrl := gomock.NewController(s.T())
	defer mockWalletCtrl.Finish()
	mockWalletService := mock_wallet_service.NewMockWalletService(mockWalletCtrl)

	// Since player lost, we expect GetOrCreateWallet to be called but not AddFunds
	mockWalletService.EXPECT().
		GetOrCreateWallet(gomock.Any(), "player1").
		Return(&entities.Wallet{UserID: "player1", Balance: 1000}, false, nil).
		Times(1)

	// Process payouts with wallet updates
	err = s.game.ProcessPayouts(context.Background(), wrapMockWalletService(mockWalletService))
	s.NoError(err)

	// Verify at least one game result was saved
	s.NotEmpty(capturedGameResults, "At least one game result should have been saved")

	// Verify all saved game results have the correct data
	for _, capturedResult := range capturedGameResults {
		s.Equal(s.game.ChannelID, capturedResult.ChannelID)
		s.Equal(entities.StateDealing, capturedResult.GameType)
		s.Equal(1, len(capturedResult.PlayerResults))

		// Verify player result details
		playerResult := capturedResult.PlayerResults[0]
		s.Equal("player1", playerResult.PlayerID)
		s.Equal(entities.Result(ResultLose), playerResult.Result)
		s.Equal(11, playerResult.Score)

		// Verify dealer details
		blackjackDetails, ok := capturedResult.Details.(*BlackjackDetails)
		s.True(ok, "Details should be of type BlackjackDetails")
		s.Equal(19, blackjackDetails.DealerScore)
		s.False(blackjackDetails.IsBlackjack)
		s.False(blackjackDetails.IsBust)
	}
}

// TestPayoutCalculation_Push tests the payout calculation when player and dealer push (tie)
func (s *GameTestSuite) TestPayoutCalculation_Push() {
	// Setup game with controlled cards for predictable results
	_ = s.game.AddPlayer("player1")

	// Create a specific deck for testing - setup for a push (tie)
	s.game.Deck = &entities.Deck{
		Cards: []*entities.Card{
			{Rank: entities.Ten, Suit: entities.Hearts},     // Player card 1
			{Rank: entities.Ten, Suit: entities.Clubs},      // Dealer card 1
			{Rank: entities.Seven, Suit: entities.Spades},   // Player card 2
			{Rank: entities.Seven, Suit: entities.Diamonds}, // Dealer card 2
			// No additional cards needed for this scenario
		},
	}

	// Deal cards manually to ensure predictable hands
	s.game.Players["player1"] = NewHand()
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[0]) // 10
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[2]) // 7 (17 total)

	s.game.Dealer = NewHand()
	s.game.Dealer.AddCard(s.game.Deck.Cards[1]) // 10
	s.game.Dealer.AddCard(s.game.Deck.Cards[3]) // 7 (17 total)

	s.game.State = entities.StatePlaying
	s.game.PlayerOrder = []string{"player1"}
	s.game.Bets["player1"] = 100

	// Make all players stand
	s.game.Players["player1"].Status = StatusStand
	s.game.Dealer.Status = StatusStand
	s.game.State = entities.StateComplete

	// Capture the game result being saved
	var capturedGameResults []*entities.GameResult
	// ProcessPayouts calls GetResults which calls SaveGameResult again
	// so we need to expect at least two calls
	s.mockGameRepo.EXPECT().
		SaveGameResult(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, result *entities.GameResult) error {
			capturedGameResults = append(capturedGameResults, result)
			return nil
		}).
		MinTimes(1)

	// SaveDeck is also called multiple times
	s.mockGameRepo.EXPECT().
		SaveDeck(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		MinTimes(1)

	// Get results - player1 should push with dealer (both have 17)
	results, err := s.game.GetResults()
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal("player1", results[0].PlayerID)
	s.Equal(ResultPush, results[0].Result)
	s.Equal(17, results[0].Score)

	// Create a mock wallet service to verify wallet updates
	mockWalletCtrl := gomock.NewController(s.T())
	defer mockWalletCtrl.Finish()
	mockWalletService := mock_wallet_service.NewMockWalletService(mockWalletCtrl)

	// In a push, player gets their original bet back
	mockWalletService.EXPECT().
		GetOrCreateWallet(gomock.Any(), "player1").
		Return(&entities.Wallet{UserID: "player1", Balance: 1000}, false, nil).
		Times(2) // Called before and after adding funds

	// Expect AddFunds to be called with the original bet amount
	mockWalletService.EXPECT().
		AddFunds(gomock.Any(), "player1", int64(100), gomock.Any()).
		Return(nil).
		Times(1)

	// Process payouts with wallet updates
	err = s.game.ProcessPayouts(context.Background(), wrapMockWalletService(mockWalletService))
	s.NoError(err)

	// Verify at least one game result was saved
	s.NotEmpty(capturedGameResults, "At least one game result should have been saved")

	// Verify all saved game results have the correct data
	for _, capturedResult := range capturedGameResults {
		s.Equal(s.game.ChannelID, capturedResult.ChannelID)
		s.Equal(entities.StateDealing, capturedResult.GameType)
		s.Equal(1, len(capturedResult.PlayerResults))

		// Verify player result details
		playerResult := capturedResult.PlayerResults[0]
		s.Equal("player1", playerResult.PlayerID)
		s.Equal(entities.Result(ResultPush), playerResult.Result)
		s.Equal(17, playerResult.Score)

		// Verify dealer details
		blackjackDetails, ok := capturedResult.Details.(*BlackjackDetails)
		s.True(ok, "Details should be of type BlackjackDetails")
		s.Equal(17, blackjackDetails.DealerScore)
		s.False(blackjackDetails.IsBlackjack)
		s.False(blackjackDetails.IsBust)
	}
}

// TestPayoutCalculation_Blackjack tests the payout calculation when player has blackjack
func (s *GameTestSuite) TestPayoutCalculation_Blackjack() {
	// Setup game with controlled cards for predictable results
	_ = s.game.AddPlayer("player1")

	// Create a specific deck for testing - setup for player blackjack
	s.game.Deck = &entities.Deck{
		Cards: []*entities.Card{
			{Rank: entities.Ace, Suit: entities.Hearts},    // Player card 1
			{Rank: entities.Ten, Suit: entities.Clubs},     // Dealer card 1
			{Rank: entities.Ten, Suit: entities.Spades},    // Player card 2
			{Rank: entities.Five, Suit: entities.Diamonds}, // Dealer card 2
			// No additional cards needed for this scenario
		},
	}

	// Deal cards manually to ensure predictable hands
	s.game.Players["player1"] = NewHand()
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[0]) // Ace
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[2]) // 10 (Blackjack: 21)

	s.game.Dealer = NewHand()
	s.game.Dealer.AddCard(s.game.Deck.Cards[1]) // 10
	s.game.Dealer.AddCard(s.game.Deck.Cards[3]) // 5 (15 total)

	s.game.State = entities.StatePlaying
	s.game.PlayerOrder = []string{"player1"}
	s.game.Bets["player1"] = 100

	// Make all players stand
	s.game.Players["player1"].Status = StatusStand
	s.game.Dealer.Status = StatusStand
	s.game.State = entities.StateComplete

	// Capture the game result being saved
	var capturedGameResults []*entities.GameResult
	// ProcessPayouts calls GetResults which calls SaveGameResult again
	// so we need to expect at least two calls
	s.mockGameRepo.EXPECT().
		SaveGameResult(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, result *entities.GameResult) error {
			capturedGameResults = append(capturedGameResults, result)
			return nil
		}).
		MinTimes(1)

	// SaveDeck is also called multiple times
	s.mockGameRepo.EXPECT().
		SaveDeck(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		MinTimes(1)

	// Get results - player1 should have blackjack
	results, err := s.game.GetResults()
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal("player1", results[0].PlayerID)
	s.Equal(ResultBlackjack, results[0].Result)
	s.Equal(21, results[0].Score)

	// Create a mock wallet service to verify wallet updates
	mockWalletCtrl := gomock.NewController(s.T())
	defer mockWalletCtrl.Finish()
	mockWalletService := mock_wallet_service.NewMockWalletService(mockWalletCtrl)

	// For blackjack, player gets 3:2 payout (bet + 1.5x bet = 250 for a 100 bet)
	mockWalletService.EXPECT().
		GetOrCreateWallet(gomock.Any(), "player1").
		Return(&entities.Wallet{UserID: "player1", Balance: 1000}, false, nil).
		Times(2) // Called before and after adding funds

	// Expect AddFunds to be called with the blackjack payout amount
	mockWalletService.EXPECT().
		AddFunds(gomock.Any(), "player1", int64(250), gomock.Any()).
		Return(nil).
		Times(1)

	// Process payouts with wallet updates
	err = s.game.ProcessPayouts(context.Background(), wrapMockWalletService(mockWalletService))
	s.NoError(err)

	// Verify at least one game result was saved
	s.NotEmpty(capturedGameResults, "At least one game result should have been saved")

	// Verify all saved game results have the correct data
	for _, capturedResult := range capturedGameResults {
		s.Equal(s.game.ChannelID, capturedResult.ChannelID)
		s.Equal(entities.StateDealing, capturedResult.GameType)
		s.Equal(1, len(capturedResult.PlayerResults))

		// Verify player result details
		playerResult := capturedResult.PlayerResults[0]
		s.Equal("player1", playerResult.PlayerID)
		s.Equal(entities.Result(ResultBlackjack), playerResult.Result)
		s.Equal(21, playerResult.Score)

		// Verify dealer details
		blackjackDetails, ok := capturedResult.Details.(*BlackjackDetails)
		s.True(ok, "Details should be of type BlackjackDetails")
		s.Equal(15, blackjackDetails.DealerScore)
		s.False(blackjackDetails.IsBlackjack)
		s.False(blackjackDetails.IsBust)
	}
}

// TestPayoutCalculation_DealerBust tests the payout calculation when dealer busts
func (s *GameTestSuite) TestPayoutCalculation_DealerBust() {
	// Setup game with controlled cards for predictable results
	_ = s.game.AddPlayer("player1")

	// Create a specific deck for testing - setup for dealer bust
	s.game.Deck = &entities.Deck{
		Cards: []*entities.Card{
			{Rank: entities.Ten, Suit: entities.Hearts},   // Player card 1
			{Rank: entities.Ten, Suit: entities.Clubs},    // Dealer card 1
			{Rank: entities.Five, Suit: entities.Spades},  // Player card 2
			{Rank: entities.Six, Suit: entities.Diamonds}, // Dealer card 2
			{Rank: entities.Jack, Suit: entities.Hearts},  // Dealer hit card (bust)
		},
	}

	// Deal cards manually to ensure predictable hands
	s.game.Players["player1"] = NewHand()
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[0]) // 10
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[2]) // 5 (15 total)

	s.game.Dealer = NewHand()
	s.game.Dealer.AddCard(s.game.Deck.Cards[1]) // 10
	s.game.Dealer.AddCard(s.game.Deck.Cards[3]) // 6 (16 total)
	s.game.Dealer.AddCard(s.game.Deck.Cards[4]) // 10 (26 total - bust)

	s.game.State = entities.StatePlaying
	s.game.PlayerOrder = []string{"player1"}
	s.game.Bets["player1"] = 100

	// Make all players stand
	s.game.Players["player1"].Status = StatusStand
	s.game.Dealer.Status = StatusStand
	s.game.State = entities.StateComplete

	// Capture the game result being saved
	var capturedGameResults []*entities.GameResult
	// ProcessPayouts calls GetResults which calls SaveGameResult again
	// so we need to expect at least two calls
	s.mockGameRepo.EXPECT().
		SaveGameResult(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, result *entities.GameResult) error {
			capturedGameResults = append(capturedGameResults, result)
			return nil
		}).
		MinTimes(1)

	// SaveDeck is also called multiple times
	s.mockGameRepo.EXPECT().
		SaveDeck(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		MinTimes(1)

	// Get results - player1 should win because dealer busts
	results, err := s.game.GetResults()
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal("player1", results[0].PlayerID)
	s.Equal(ResultWin, results[0].Result)
	s.Equal(15, results[0].Score)

	// Create a mock wallet service to verify wallet updates
	mockWalletCtrl := gomock.NewController(s.T())
	defer mockWalletCtrl.Finish()
	mockWalletService := mock_wallet_service.NewMockWalletService(mockWalletCtrl)

	// For regular win, player gets 1:1 payout (bet + equal amount = 200 for a 100 bet)
	mockWalletService.EXPECT().
		GetOrCreateWallet(gomock.Any(), "player1").
		Return(&entities.Wallet{UserID: "player1", Balance: 1000}, false, nil).
		Times(2) // Called before and after adding funds

	// Expect AddFunds to be called with the regular win payout amount
	mockWalletService.EXPECT().
		AddFunds(gomock.Any(), "player1", int64(200), gomock.Any()).
		Return(nil).
		Times(1)

	// Process payouts with wallet updates
	err = s.game.ProcessPayouts(context.Background(), wrapMockWalletService(mockWalletService))
	s.NoError(err)

	// Verify at least one game result was saved
	s.NotEmpty(capturedGameResults, "At least one game result should have been saved")

	// Verify all saved game results have the correct data
	for _, capturedResult := range capturedGameResults {
		s.Equal(s.game.ChannelID, capturedResult.ChannelID)
		s.Equal(entities.StateDealing, capturedResult.GameType)
		s.Equal(1, len(capturedResult.PlayerResults))

		// Verify player result details
		playerResult := capturedResult.PlayerResults[0]
		s.Equal("player1", playerResult.PlayerID)
		s.Equal(entities.Result(ResultWin), playerResult.Result)
		s.Equal(15, playerResult.Score)

		// Verify dealer details
		blackjackDetails, ok := capturedResult.Details.(*BlackjackDetails)
		s.True(ok, "Details should be of type BlackjackDetails")
		s.Equal(26, blackjackDetails.DealerScore)
		s.False(blackjackDetails.IsBlackjack)
		s.True(blackjackDetails.IsBust)
	}
}

// TestBlackjackPayout tests the payout calculation for blackjack
func (s *GameTestSuite) TestBlackjackPayout() {
	// Setup game with controlled cards for predictable results
	_ = s.game.AddPlayer("player1")

	// Create a specific deck for testing - setup for a blackjack
	s.game.Deck = &entities.Deck{
		Cards: []*entities.Card{
			{Rank: entities.Ace, Suit: entities.Hearts},    // Player card 1
			{Rank: entities.Ten, Suit: entities.Clubs},     // Dealer card 1
			{Rank: entities.Ten, Suit: entities.Spades},    // Player card 2
			{Rank: entities.Five, Suit: entities.Diamonds}, // Dealer card 2
		},
	}

	// Deal cards manually to ensure predictable hands
	s.game.Players["player1"] = NewHand()
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[0]) // Ace
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[2]) // 10 (blackjack)

	s.game.Dealer = NewHand()
	s.game.Dealer.AddCard(s.game.Deck.Cards[1]) // 10
	s.game.Dealer.AddCard(s.game.Deck.Cards[3]) // 5 (15 total)

	s.game.State = entities.StatePlaying
	s.game.PlayerOrder = []string{"player1"}
	s.game.Bets["player1"] = 100

	// Make all players stand
	s.game.Players["player1"].Status = StatusStand
	s.game.Dealer.Status = StatusStand
	s.game.State = entities.StateComplete

	// Capture the game result being saved
	var capturedGameResults []*entities.GameResult
	// ProcessPayouts calls GetResults which calls SaveGameResult again
	// so we need to expect at least two calls
	s.mockGameRepo.EXPECT().
		SaveGameResult(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, result *entities.GameResult) error {
			capturedGameResults = append(capturedGameResults, result)
			return nil
		}).
		MinTimes(1)

	// SaveDeck is also called multiple times
	s.mockGameRepo.EXPECT().
		SaveDeck(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		MinTimes(1)

	// Get results - player1 should win with blackjack
	results, err := s.game.GetResults()
	s.NoError(err)
	s.Equal(1, len(results))
	s.Equal("player1", results[0].PlayerID)
	s.Equal(ResultBlackjack, results[0].Result)
	s.Equal(21, results[0].Score)

	// Create a mock wallet service to verify wallet updates
	mockWalletCtrl := gomock.NewController(s.T())
	defer mockWalletCtrl.Finish()
	mockWalletService := mock_wallet_service.NewMockWalletService(mockWalletCtrl)

	// For blackjack, player gets 3:2 payout (bet + 1.5x bet = 250 for a 100 bet)
	mockWalletService.EXPECT().
		GetOrCreateWallet(gomock.Any(), "player1").
		Return(&entities.Wallet{UserID: "player1", Balance: 1000}, false, nil).
		Times(2) // Called before and after adding funds

	// Expect AddFunds to be called with the blackjack payout amount
	mockWalletService.EXPECT().
		AddFunds(gomock.Any(), "player1", int64(250), gomock.Any()).
		Return(nil).
		Times(1)

	// Process payouts with wallet updates
	err = s.game.ProcessPayouts(context.Background(), wrapMockWalletService(mockWalletService))
	s.NoError(err)

	// Verify at least one game result was saved
	s.NotEmpty(capturedGameResults, "At least one game result should have been saved")

	// Verify all saved game results have the correct data
	for _, capturedResult := range capturedGameResults {
		s.Equal(s.game.ChannelID, capturedResult.ChannelID)
		s.Equal(entities.StateDealing, capturedResult.GameType)
		s.Equal(1, len(capturedResult.PlayerResults))

		// Verify player result details
		playerResult := capturedResult.PlayerResults[0]
		s.Equal("player1", playerResult.PlayerID)
		s.Equal(entities.Result(ResultBlackjack), playerResult.Result)
		s.Equal(21, playerResult.Score)

		// Verify dealer details
		blackjackDetails, ok := capturedResult.Details.(*BlackjackDetails)
		s.True(ok, "Details should be of type BlackjackDetails")
		s.Equal(15, blackjackDetails.DealerScore)
		s.False(blackjackDetails.IsBlackjack)
		s.False(blackjackDetails.IsBust)
	}
}

// TestGameCompletionWithBust tests game completion when a player busts
func (s *GameTestSuite) TestGameCompletionWithBust() {
	// Setup game with multiple players where one player busts
	_ = s.game.AddPlayer("player1")
	_ = s.game.AddPlayer("player2")

	// Create a specific deck for testing
	s.game.Deck = &entities.Deck{
		Cards: []*entities.Card{
			{Rank: entities.Ten, Suit: entities.Hearts},    // Player1 card 1
			{Rank: entities.Nine, Suit: entities.Clubs},    // Player2 card 1
			{Rank: entities.Ten, Suit: entities.Spades},    // Dealer card 1
			{Rank: entities.Seven, Suit: entities.Hearts},  // Player1 card 2
			{Rank: entities.Eight, Suit: entities.Clubs},   // Player2 card 2
			{Rank: entities.Seven, Suit: entities.Spades},  // Dealer card 2
			{Rank: entities.King, Suit: entities.Diamonds}, // Player2 hit card (bust)
		},
	}

	// Deal cards manually to ensure predictable hands
	s.game.Players["player1"] = NewHand()
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[0]) // 10
	s.game.Players["player1"].AddCard(s.game.Deck.Cards[3]) // 7

	s.game.Players["player2"] = NewHand()
	s.game.Players["player2"].AddCard(s.game.Deck.Cards[1]) // 9
	s.game.Players["player2"].AddCard(s.game.Deck.Cards[4]) // 8
	s.game.Players["player2"].AddCard(s.game.Deck.Cards[6]) // K (bust with 27)
	s.game.Players["player2"].Status = StatusBust

	s.game.Dealer = NewHand()
	s.game.Dealer.AddCard(s.game.Deck.Cards[2]) // 10
	s.game.Dealer.AddCard(s.game.Deck.Cards[5]) // 7

	s.game.State = entities.StatePlaying
	s.game.PlayerOrder = []string{"player1", "player2"}

	// Player1 is standing, player2 is bust
	s.game.Players["player1"].Status = StatusStand
	s.game.Dealer.Status = StatusStand
	s.game.State = entities.StateComplete

	// Verify all players are done
	s.True(s.game.CheckAllPlayersDone(), "All players should be done when one stands and one busts")

	// Mock the SaveGameResult and SaveDeck calls for GetResults
	s.mockGameRepo.EXPECT().
		SaveGameResult(gomock.Any(), gomock.Any()).
		Return(nil)

	s.mockGameRepo.EXPECT().
		SaveDeck(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	// Get results
	results, err := s.game.GetResults()
	s.NoError(err)
	s.Equal(2, len(results))

	// Check results for both players without assuming order
	var player1Result, player2Result *HandResult
	for i := range results {
		if results[i].PlayerID == "player1" {
			player1Result = &results[i]
		} else if results[i].PlayerID == "player2" {
			player2Result = &results[i]
		}
	}

	// Player1 should push with dealer (both have 17)
	s.NotNil(player1Result, "Player1 result should be present")
	s.Equal(ResultPush, player1Result.Result)
	s.Equal(17, player1Result.Score)

	// Player2 should lose (bust)
	s.NotNil(player2Result, "Player2 result should be present")
	s.Equal(ResultLose, player2Result.Result)
}

// mockWalletServiceWrapper wraps a mock wallet service and adds the GetStandardLoanIncrement method
type mockWalletServiceWrapper struct {
	*mock_wallet_service.MockWalletService
}

// GetStandardLoanIncrement implements the WalletService interface
func (m *mockWalletServiceWrapper) GetStandardLoanIncrement() int64 {
	return 100 // Return the standard loan amount for tests
}

// CanRepayLoan implements the WalletService interface
func (m *mockWalletServiceWrapper) CanRepayLoan(ctx context.Context, userID string) (bool, error) {
	// For testing purposes, assume a loan can be repaid if userID contains "loan"
	return strings.Contains(userID, "loan"), nil
}

// wrapMockWalletService wraps a mock wallet service to implement the full WalletService interface
func wrapMockWalletService(mock *mock_wallet_service.MockWalletService) WalletService {
	return &mockWalletServiceWrapper{MockWalletService: mock}
}
