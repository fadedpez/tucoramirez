package entities

import "time"

// Base result types
type Result string

const (
	ResultWin  Result = "WIN"
	ResultLose Result = "LOSE"
	ResultPush Result = "PUSH"
)

// Game state types
type GameState string

const (
	StateWaiting  GameState = "WAITING"
	StateDealing  GameState = "DEALING"
	StatePlaying  GameState = "PLAYING"
	StateDealer   GameState = "DEALER"
	StateComplete GameState = "COMPLETE"
)

// GameDetails defines what game-specific result details must provide
type GameDetails interface {
	// GameType returns the type of game (ex blackjack or poker)
	GameType() GameState
	// ValidateDetails ensures the details are valid for the game
	ValidateDetails() error
}

// GameResult represents the outcome of any game
type GameResult struct {
	ChannelID     string
	GameType      GameState
	CompletedAt   time.Time
	PlayerResults []*PlayerResult
	Details       GameDetails
}

type PlayerResult struct {
	PlayerID string
	Result   Result
	Score    int
}
