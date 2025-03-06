package types

import "fmt"

// ErrorCode represents a specific error type
type ErrorCode string

const (
	// Game state errors
	ErrGameNotFound    ErrorCode = "GAME_NOT_FOUND"
	ErrGameInProgress  ErrorCode = "GAME_IN_PROGRESS"
	ErrGameAlreadyEnded ErrorCode = "GAME_ALREADY_ENDED"
	ErrInvalidState    ErrorCode = "INVALID_STATE"

	// Player errors
	ErrPlayerNotFound   ErrorCode = "PLAYER_NOT_FOUND"
	ErrNotPlayerTurn    ErrorCode = "NOT_PLAYER_TURN"
	ErrNotGameCreator   ErrorCode = "NOT_GAME_CREATOR"
	ErrAlreadyJoined    ErrorCode = "ALREADY_JOINED"
	ErrTooManyPlayers   ErrorCode = "TOO_MANY_PLAYERS"
	ErrNotEnoughPlayers ErrorCode = "NOT_ENOUGH_PLAYERS"
	ErrPlayerBusted     ErrorCode = "PLAYER_BUSTED"
	ErrPlayerStanding   ErrorCode = "PLAYER_STANDING"

	// Action errors
	ErrInvalidAction    ErrorCode = "INVALID_ACTION"
	ErrInvalidCommand   ErrorCode = "INVALID_COMMAND"
	ErrInvalidArgument  ErrorCode = "INVALID_ARGUMENT"
	ErrPermissionDenied ErrorCode = "PERMISSION_DENIED"

	// System errors
	ErrInternalError    ErrorCode = "INTERNAL_ERROR"
	ErrNetworkError     ErrorCode = "NETWORK_ERROR"
	ErrDatabaseError    ErrorCode = "DATABASE_ERROR"
	ErrRateLimited      ErrorCode = "RATE_LIMITED"
)

// GameError represents a game-related error
type GameError struct {
	Code    ErrorCode
	Message string
	Err     error // Underlying error, if any
}

// Error implements the error interface
func (e *GameError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *GameError) Unwrap() error {
	return e.Err
}

// NewGameError creates a new GameError
func NewGameError(code ErrorCode, message string) *GameError {
	return &GameError{
		Code:    code,
		Message: message,
	}
}

// WrapError wraps an existing error in a GameError
func WrapError(code ErrorCode, message string, err error) *GameError {
	return &GameError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsGameError checks if an error is a GameError and has a specific code
func IsGameError(err error, code ErrorCode) bool {
	var gameErr *GameError
	if err == nil {
		return false
	}
	if ok := As(err, &gameErr); !ok {
		return false
	}
	return gameErr.Code == code
}

// As is a helper function to safely type assert an error to a GameError
func As(err error, target **GameError) bool {
	if target == nil {
		return false
	}
	if gameErr, ok := err.(*GameError); ok {
		*target = gameErr
		return true
	}
	return false
}
