package utils

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// GameError represents a game-related error
type GameError struct {
	Message string
	Code    string
}

func (e *GameError) Error() string {
	return e.Message
}

// NewGameError creates a new GameError
func NewGameError(code, message string) *GameError {
	return &GameError{
		Message: message,
		Code:    code,
	}
}

// Common error codes
const (
	ErrGameInProgress    = "GAME_IN_PROGRESS"
	ErrGameNotFound      = "GAME_NOT_FOUND"
	ErrPlayerNotFound    = "PLAYER_NOT_FOUND"
	ErrTooManyPlayers    = "TOO_MANY_PLAYERS"
	ErrNotEnoughPlayers  = "NOT_ENOUGH_PLAYERS"
	ErrNotPlayerTurn     = "NOT_PLAYER_TURN"
	ErrGameAlreadyEnded  = "GAME_ALREADY_ENDED"
)

// SendErrorResponse sends an ephemeral error message to the user
func SendErrorResponse(s *discordgo.Session, i *discordgo.InteractionCreate, err error) {
	var message string
	if gameErr, ok := err.(*GameError); ok {
		message = gameErr.Message
	} else {
		message = "An unexpected error occurred"
		fmt.Printf("Unexpected error: %v\n", err)
	}

	response := &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}

	if err := s.InteractionRespond(i.Interaction, response); err != nil {
		fmt.Printf("Error sending error response: %v\n", err)
	}
}
