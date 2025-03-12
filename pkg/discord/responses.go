package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/pkg/services/blackjack"
)

// respondWithError sends an error message as an ephemeral response
func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, message string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "¡Ay, caramba! " + message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// createGameEmbed creates the message embed showing the game state
func createGameEmbed(game *blackjack.Game, s *discordgo.Session, guildID string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: "¡Blackjack con Tuco!",
		Color: 0xFFD700, // Gold color for that bandit style
	}

	// Add dealer's hand first
	dealerField := createDealerField(game)
	embed.Fields = append(embed.Fields, dealerField)

	// Add all players' hands
	for playerID, hand := range game.Players {
		playerScore := blackjack.GetBestScore(hand.Cards)
		playerStatus := getStatusMessage(hand.Status)

		// Get member info for display name
		member, err := s.GuildMember(guildID, playerID)
		playerName := "Unknown Player"
		if err == nil && member.Nick != "" {
			playerName = member.Nick
		} else if err == nil && member.User != nil {
			playerName = member.User.Username
		}

		if hand.Status == blackjack.StatusPlaying {
			playerName += " 🎲" // Current player indicator
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("💎 %s", playerName),
			Value:  fmt.Sprintf("%s\nScore: %d%s", FormatCards(hand.Cards), playerScore, playerStatus),
			Inline: true,
		})
	}

	// Add game result message if game is complete
	if game.State == blackjack.StateComplete {
		embed.Description = getGameResultsDescription(game, s, guildID)
	}

	return embed
}

// getGameResultsDescription returns a summary of all players' results
func getGameResultsDescription(game *blackjack.Game, s *discordgo.Session, guildID string) string {
	dealerScore := blackjack.GetBestScore(game.Dealer.Cards)
	var results string

	if dealerScore > 21 {
		results = "¡MADRE DE DIOS! Tuco went bust! Everyone still standing wins! 💰💰💰\n\n"
	} else {
		results = fmt.Sprintf("¡El Dealer tiene %d! Let's see who won...\n\n", dealerScore)
	}

	for playerID, hand := range game.Players {
		playerScore := blackjack.GetBestScore(hand.Cards)
		playerResult := ""

		switch {
		case hand.Status == blackjack.StatusBust:
			playerResult = " 💥 ¡BUST!"
		case dealerScore > 21:
			playerResult = " 💰 ¡GANADOR!"
		case playerScore > dealerScore:
			playerResult = " 💰 ¡GANADOR!"
		case playerScore < dealerScore:
			playerResult = " 💔 ¡PERDEDOR!"
		case playerScore == dealerScore:
			playerResult = ":beers: ¡EMPATE!"
		}

		// Get member info for display name
		member, err := s.GuildMember(guildID, playerID)
		playerName := "Unknown Player"
		if err == nil && member.Nick != "" {
			playerName = member.Nick
		} else if err == nil && member.User != nil {
			playerName = member.User.Username
		}

		results += fmt.Sprintf("**%s**: %s (%d)\n", playerName, playerResult, playerScore)
	}

	return results
}

// createGameButtons creates the action buttons if the game is in progress
func createGameButtons(game *blackjack.Game) []discordgo.MessageComponent {
	if game.State != blackjack.StatePlaying {
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Play Again",
						Style:    discordgo.PrimaryButton,
						CustomID: "play_again",
					},
				},
			},
		}
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Hit",
					Style:    discordgo.PrimaryButton,
					CustomID: "hit",
				},
				discordgo.Button{
					Label:    "Stand",
					Style:    discordgo.SecondaryButton,
					CustomID: "stand",
				},
			},
		},
	}
}

// createDealerField creates the dealer's hand field
func createDealerField(game *blackjack.Game) *discordgo.MessageEmbedField {
	dealerScore := blackjack.GetBestScore(game.Dealer.Cards)
	dealerStatus := getStatusMessage(game.Dealer.Status)

	var dealerValue string
	if game.State == blackjack.StateComplete {
		// Show all cards at the end of the game
		dealerValue = fmt.Sprintf("%s\nScore: %d%s", FormatCards(game.Dealer.Cards), dealerScore, dealerStatus)
	} else {
		// During play, only show first card and hide the rest
		dealerValue = fmt.Sprintf("%s 🎴\nScore: ?", FormatCard(game.Dealer.Cards[0]))
	}

	return &discordgo.MessageEmbedField{
		Name:   "🎩 El Dealer (Tuco)",
		Value:  dealerValue,
		Inline: true,
	}
}

// getStatusMessage returns a status message based on hand status
func getStatusMessage(status blackjack.Status) string {
	switch status {
	case blackjack.StatusBust:
		return " 💥 ¡BUST!"
	case blackjack.StatusStand:
		return " 🛑 ¡STAND!"
	default:
		return ""
	}
}

// createLobbyEmbed creates the message embed for the lobby display
func createLobbyEmbed(lobby *GameLobby) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "¡Bienvenidos a la Mesa de Tuco!",
		Description: "*Tuco polishes his golden rings while waiting for players*\n\n¡Siéntate, amigo! Take a seat at my table! 🎰",
		Color:       0xFFD700, // Gold color
	}

	// Add player list
	playerList := ""
	for playerID, joined := range lobby.Players {
		if !joined {
			continue
		}
		if playerID == lobby.OwnerID {
			playerList += fmt.Sprintf("<@%s> (El Jefe)\n", playerID)
		} else {
			playerList += fmt.Sprintf("<@%s>\n", playerID)
		}
	}

	if playerList == "" {
		playerList = "No players yet... ¡Qué triste!"
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "🎲 Players at the Table",
		Value:  playerList,
		Inline: false,
	})

	return embed
}

// createLobbyButtons creates the join and start buttons for the lobby
func createLobbyButtons(ownerID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: "join_game",
					Label:    "Join",
					Style:    discordgo.PrimaryButton,
				},
				discordgo.Button{
					CustomID: "start_game",
					Label:    "Start",
					Style:    discordgo.SuccessButton,
				},
			},
		},
	}
}
