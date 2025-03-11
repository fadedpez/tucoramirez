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

// displayGameState shows the current game state with cards and buttons
func (b *Bot) displayGameState(s *discordgo.Session, i *discordgo.InteractionCreate, game *blackjack.Game) {
	embed := createGameEmbed(game, i.Member.User.ID)
	components := createGameButtons(game)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	})
}

// createGameEmbed creates the message embed showing the game state
func createGameEmbed(game *blackjack.Game, playerID string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: "¡Blackjack con Tuco!",
		Color: 0xFFD700, // Gold color for that bandit style
	}

	// Add player's hand
	playerHand := game.Players[playerID]
	playerScore := blackjack.GetBestScore(playerHand.Cards)
	playerStatus := getStatusMessage(playerHand.Status)

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "💎 Tu mano (Your Hand)",
		Value:  fmt.Sprintf("%s\nScore: %d%s", formatCards(playerHand.Cards), playerScore, playerStatus),
		Inline: true,
	})

	// Add dealer's hand
	dealerField := createDealerField(game)
	embed.Fields = append(embed.Fields, dealerField)

	return embed
}

// createGameButtons creates the action buttons if the game is in progress
func createGameButtons(game *blackjack.Game) []discordgo.MessageComponent {
	if game.State != blackjack.StatePlaying {
		return nil
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					CustomID: "hit",
					Label:    "¡Dame una carta! (Hit)",
					Style:    discordgo.PrimaryButton,
					Emoji: &discordgo.ComponentEmoji{
						Name: "🎴",
					},
				},
				discordgo.Button{
					CustomID: "stand",
					Label:    "¡Me planto! (Stand)",
					Style:    discordgo.SecondaryButton,
					Emoji: &discordgo.ComponentEmoji{
						Name: "✋",
					},
				},
			},
		},
	}
}

// createDealerField creates the dealer's hand field
func createDealerField(game *blackjack.Game) *discordgo.MessageEmbedField {
	var dealerValue string
	if game.State == blackjack.StateComplete {
		dealerScore := blackjack.GetBestScore(game.Dealer.Cards)
		dealerValue = fmt.Sprintf("%s\nScore: %d", formatCards(game.Dealer.Cards), dealerScore)
	} else {
		// Hide second card during play
		dealerValue = fmt.Sprintf("%s 🎴", formatCard(game.Dealer.Cards[0]))
	}

	return &discordgo.MessageEmbedField{
		Name:   "🎩 La mano del dealer",
		Value:  dealerValue,
		Inline: true,
	}
}

// getStatusMessage returns a status message based on hand status
func getStatusMessage(status blackjack.Status) string {
	switch status {
	case blackjack.StatusBust:
		return "\n¡Ay, te pasaste! (Bust!)"
	case blackjack.StatusStand:
		return "\n¡Te plantas! (Standing)"
	default:
		return ""
	}
}
