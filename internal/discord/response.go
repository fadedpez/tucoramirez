package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/types"
)

// ResponseEmoji maps error codes to appropriate emojis
var ResponseEmoji = map[types.ErrorCode]string{
	types.ErrGameNotFound:    "🔍",
	types.ErrGameInProgress:  "🎮",
	types.ErrGameAlreadyEnded: "🏁",
	types.ErrInvalidState:    "⚠️",
	types.ErrPlayerNotFound:   "👤",
	types.ErrNotPlayerTurn:    "⏳",
	types.ErrNotGameCreator:   "👑",
	types.ErrAlreadyJoined:    "✋",
	types.ErrTooManyPlayers:   "👥",
	types.ErrNotEnoughPlayers: "🤷",
	types.ErrInvalidAction:    "❌",
	types.ErrInvalidCommand:   "⛔",
	types.ErrInvalidArgument:  "❗",
	types.ErrPermissionDenied: "🚫",
	types.ErrInternalError:    "💥",
	types.ErrNetworkError:     "🌐",
	types.ErrDatabaseError:    "💾",
	types.ErrRateLimited:      "⏱️",
}

// Response represents a Discord interaction response
type Response struct {
	Content    string
	Components []discordgo.MessageComponent
	Ephemeral  bool
}

// NewResponse creates a new Response
func NewResponse(content string, components []discordgo.MessageComponent) *Response {
	return &Response{
		Content:    content,
		Components: components,
		Ephemeral:  false,
	}
}

// NewEphemeralResponse creates a new ephemeral Response (only visible to the user)
func NewEphemeralResponse(content string, components []discordgo.MessageComponent) *Response {
	return &Response{
		Content:    content,
		Components: components,
		Ephemeral:  true,
	}
}

// NewErrorResponse creates a new error Response
func NewErrorResponse(err error) *Response {
	var gameErr *types.GameError
	if types.As(err, &gameErr) {
		emoji := ResponseEmoji[gameErr.Code]
		if emoji == "" {
			emoji = "❌"
		}
		return NewEphemeralResponse(fmt.Sprintf("%s %s", emoji, gameErr.Message), nil)
	}
	return NewEphemeralResponse(fmt.Sprintf("❌ An error occurred: %v", err), nil)
}

// SendResponse sends a response to a Discord interaction
func SendResponse(s SessionHandler, i *discordgo.InteractionCreate, r *Response) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    r.Content,
			Components: r.Components,
			Flags:      getFlags(r.Ephemeral),
		},
	})
}

// UpdateResponse updates an existing interaction response
func UpdateResponse(s SessionHandler, i *discordgo.InteractionCreate, r *Response) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    r.Content,
			Components: r.Components,
			Flags:      getFlags(r.Ephemeral),
		},
	})
}

// SendGameResponse sends a game response
func SendGameResponse(s SessionHandler, i *discordgo.InteractionCreate, content string, components []discordgo.MessageComponent) error {
	return SendResponse(s, i, NewResponse(content, components))
}

// UpdateGameResponse updates a game response
func UpdateGameResponse(s SessionHandler, i *discordgo.InteractionCreate, content string, components []discordgo.MessageComponent) error {
	return UpdateResponse(s, i, NewResponse(content, components))
}

// SendErrorResponse sends an error response
func SendErrorResponse(s SessionHandler, i *discordgo.InteractionCreate, err error) error {
	return SendResponse(s, i, NewErrorResponse(err))
}

// Helper functions

func getFlags(ephemeral bool) discordgo.MessageFlags {
	if ephemeral {
		return discordgo.MessageFlagsEphemeral
	}
	return 0
}
