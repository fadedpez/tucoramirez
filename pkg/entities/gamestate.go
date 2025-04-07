package entities

import "time"

// Result represents the outcome of a player's participation in a game
type Result interface {
	// String returns the string representation of the result
	String() string
	
	// IsWin returns true if this result represents a win
	IsWin() bool
}

// StringResult is a simple string-based implementation of Result
type StringResult string

// String returns the string representation of the result
func (r StringResult) String() string {
	return string(r)
}

// IsWin returns true if this result represents a win
func (r StringResult) IsWin() bool {
	return r == StringResultWin || r == StringResultBlackjack
}

// Common result constants
const (
	StringResultWin       StringResult = "WIN"
	StringResultLose      StringResult = "LOSE"
	StringResultPush      StringResult = "PUSH"
	StringResultBlackjack StringResult = "BLACKJACK"
)

// Game state types
type GameState string

const (
	StateWaiting   GameState = "WAITING"
	StateBetting   GameState = "BETTING"
	StateDealing   GameState = "DEALING"
	StateSpecialBet GameState = "SPECIAL_BET"
	StatePlaying   GameState = "PLAYING"
	StateDealer    GameState = "DEALER"
	StateComplete  GameState = "COMPLETE"
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
