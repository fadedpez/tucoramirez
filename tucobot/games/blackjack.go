package games

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot/cards"
)

// Interface for Discord session operations
type sessionHandler interface {
	// Interaction responses
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	// Message sending
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

var activeGames = make(map[string]*cards.GameSession)

func StartBlackjackGame(s sessionHandler, channelID string) error {
	game := cards.NewGame()
	activeGames[channelID] = game

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "A new game of blackjack has started! Click Join to play.",
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Join",
						CustomID: "join_button",
						Style:    discordgo.PrimaryButton,
					},
					discordgo.Button{
						Label:    "Deal",
						CustomID: "deal_button",
						Style:    discordgo.SuccessButton,
					},
				},
			},
		},
	})
	return err
}

func HandleButton(s sessionHandler, i *discordgo.InteractionCreate) error {
	game, exists := activeGames[i.ChannelID]
	if !exists {
		return fmt.Errorf("no active game in this channel")
	}

	var response string
	var err error

	switch i.MessageComponentData().CustomID {
	case "join_button":
		response, err = handleJoinButton(game, i)
	case "deal_button":
		response, err = handleDealButton(game, i)
	case "hit_button":
		response, err = handleHitButton(game, i)
	case "stand_button":
		response, err = handleStandButton(game, i)
	default:
		err = fmt.Errorf("unknown button: %s", i.MessageComponentData().CustomID)
	}

	if err != nil {
		return respondWithError(s, i, err)
	}

	return respondWithGameState(s, i, game, response)
}

func handleJoinButton(game *cards.GameSession, i *discordgo.InteractionCreate) (string, error) {
	player := &cards.Player{
		ID:   i.Member.User.ID,
		Name: i.Member.User.Username,
	}

	for _, p := range game.Players {
		if p.ID == player.ID {
			return "", fmt.Errorf("you have already joined the game")
		}
	}

	game.Players = append(game.Players, *player)
	return fmt.Sprintf("%s has joined the game!", player.Name), nil
}

func handleDealButton(game *cards.GameSession, i *discordgo.InteractionCreate) (string, error) {
	if len(game.Players) == 0 {
		return "", fmt.Errorf("no players have joined yet")
	}

	// Deal two cards to each player
	for _, player := range game.Players {
		game.DealCard(player.ID)
		game.DealCard(player.ID)
	}

	// Deal two cards to dealer
	game.DealCard("dealer")
	game.DealCard("dealer")

	return "Cards have been dealt!", nil
}

func handleHitButton(game *cards.GameSession, i *discordgo.InteractionCreate) (string, error) {
	var player *cards.Player
	for j := range game.Players {
		if game.Players[j].ID == i.Member.User.ID {
			player = &game.Players[j]
			break
		}
	}

	if player == nil {
		return "", fmt.Errorf("you are not in this game")
	}

	game.DealCard(player.ID)

	if player.Score > 21 {
		return fmt.Sprintf("%s busts with %d!", player.Name, player.Score), nil
	}

	return fmt.Sprintf("%s hits!", player.Name), nil
}

func handleStandButton(game *cards.GameSession, i *discordgo.InteractionCreate) (string, error) {
	var player *cards.Player
	for j := range game.Players {
		if game.Players[j].ID == i.Member.User.ID {
			player = &game.Players[j]
			break
		}
	}

	if player == nil {
		return "", fmt.Errorf("you are not in this game")
	}

	// Dealer's turn
	for game.Dealer.Score < 17 {
		game.DealCard("dealer")
	}

	winner := game.DetermineWinner()
	var result string

	switch winner {
	case "dealer":
		result = "Dealer wins!"
	case "player":
		result = fmt.Sprintf("%s wins!", player.Name)
	case "tie":
		result = "It's a tie!"
	}

	delete(activeGames, i.ChannelID)
	return result, nil
}

func respondWithError(s sessionHandler, i *discordgo.InteractionCreate, err error) error {
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Error: %v", err),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func respondWithGameState(s sessionHandler, i *discordgo.InteractionCreate, game *cards.GameSession, message string) error {
	var content strings.Builder
	content.WriteString(message)
	content.WriteString("\n\n")
	content.WriteString(game.GetGameState())

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content.String(),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Hit",
							CustomID: "hit_button",
							Style:    discordgo.PrimaryButton,
						},
						discordgo.Button{
							Label:    "Stand",
							CustomID: "stand_button",
							Style:    discordgo.DangerButton,
						},
					},
				},
			},
		},
	})
}
