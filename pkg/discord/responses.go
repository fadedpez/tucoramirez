package discord

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/fadedpez/tucoramirez/pkg/services/blackjack"
)

// SessionInterface defines the methods needed from a Discord session
type SessionInterface interface {
	GuildMember(guildID, userID string, options ...discordgo.RequestOption) (*discordgo.Member, error)
}

// createGameEmbed creates the message embed showing the game state
func createGameEmbed(game *blackjack.Game, s SessionInterface, guildID string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title: "¡Blackjack con Tuco!",
		Color: 0xFFD700, // Gold color for that bandit style
	}

	// Add dealer's hand first
	dealerField := createDealerField(game)
	embed.Fields = append(embed.Fields, dealerField)

	// Get current player's turn if game is in playing state
	var currentPlayerID string
	if game.State == entities.StatePlaying {
		currentPlayer, err := game.GetCurrentTurnPlayerID()
		if err == nil {
			currentPlayerID = currentPlayer
		}
	}

	// Add all players' hands
	// Iterate through PlayerOrder to maintain consistent display order
	for _, playerID := range game.PlayerOrder {
		hand := game.Players[playerID]

		// Don't skip parent hands of splits anymore - we want to show them

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

		// Add bet amount if available
		if bet, hasBet := game.Bets[playerID]; hasBet {
			playerName = fmt.Sprintf("%s (Bet: $%d)", playerName, bet)
		}

		// Add turn indicator if it's this player's turn
		var namePrefix string
		if (game.State == entities.StatePlaying && playerID == currentPlayerID) ||
			(game.State == entities.StateBetting && len(game.PlayerOrder) > 0 &&
				game.CurrentBettingPlayer < len(game.PlayerOrder) &&
				game.PlayerOrder[game.CurrentBettingPlayer] == playerID) ||
			(game.State == "SPECIAL_BETS" && len(game.PlayerOrder) > 0 &&
				game.CurrentSpecialBetsTurn < len(game.PlayerOrder) &&
				game.PlayerOrder[game.CurrentSpecialBetsTurn] == playerID) {
			namePrefix = "👉 " // Pointing finger emoji to indicate current turn
		} else {
			namePrefix = "" // No emoji for other players
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s%s", namePrefix, playerName),
			Value:  fmt.Sprintf("%s\nScore: %d%s", FormatCards(hand.Cards), playerScore, playerStatus),
			Inline: true,
		})
	}

	// Add split hands
	for handID, hand := range game.Players {
		// Skip non-split hands (they're already displayed above)
		if !hand.IsSplit() {
			continue
		}

		// Get the parent player ID
		parentID := hand.GetParentHandID()
		if parentID == "" {
			continue // Skip if no parent ID (shouldn't happen for valid split hands)
		}

		// Skip the "Unknown Player" split hand
		// The split hand ID is in the format "playerID_split"
		if handID != parentID+"_split" {
			continue
		}

		// Get member info for display name
		member, err := s.GuildMember(guildID, parentID)
		playerName := "Unknown Player"
		if err == nil && member.Nick != "" {
			playerName = member.Nick
		} else if err == nil && member.User != nil {
			playerName = member.User.Username
		}
		playerName = fmt.Sprintf("%s (Split)", playerName)

		// Add bet amount if available
		if bet, hasBet := game.Bets[handID]; hasBet {
			playerName = fmt.Sprintf("%s (Bet: $%d)", playerName, bet)
		}

		// Calculate score and status
		playerScore := blackjack.GetBestScore(hand.Cards)
		playerStatus := getStatusMessage(hand.Status)

		// Add turn indicator if it's this hand's turn
		var namePrefix string
		if game.State == entities.StatePlaying && handID == currentPlayerID {
			namePrefix = "👉 " // Pointing finger emoji to indicate current turn
		} else {
			namePrefix = "" // No emoji for other hands
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s%s", namePrefix, playerName),
			Value:  fmt.Sprintf("%s\nScore: %d%s", FormatCards(hand.Cards), playerScore, playerStatus),
			Inline: true,
		})
	}

	// Add game result message if game is complete
	if game.State == entities.StateComplete {
		embed.Description = getGameResultsDescription(game, s, guildID)

		// Add random image from image service if available
		// if bot != nil && bot.imageService != nil {
		// 	image := bot.imageService.GetRandomImage()
		// 	if image != nil && image.URL != "" {
		// 		embed.Image = &discordgo.MessageEmbedImage{
		// 			URL: image.URL,
		// 		}
		// 	}
		// }
	}

	return embed
}

// getGameResultsDescription returns a summary of all players' results
func getGameResultsDescription(game *blackjack.Game, s SessionInterface, guildID string) string {
	dealerScore := blackjack.GetBestScore(game.Dealer.Cards)
	var results string

	if dealerScore > 21 {
		results = "¡MADRE DE DIOS! Tuco went bust! Everyone still standing wins! 💰💰💰\n\n"
	} else {
		results = fmt.Sprintf("¡El Dealer tiene %d! Let's see who won...\n\n", dealerScore)
	}

	// Calculate payouts for each player
	payouts := game.CalculatePayouts()

	for playerID, hand := range game.Players {
		playerScore := blackjack.GetBestScore(hand.Cards)
		playerResult := ""
		bet := game.Bets[playerID]
		payout := payouts[playerID]

		// For blackjack, the payout already includes the correct amount (bet + winnings)
		// For other results, we need to calculate the net result differently
		var netResult int64

		switch {
		case hand.Status == blackjack.StatusBust:
			playerResult = " 💥 ¡BUST!"
			netResult = -bet // Loss equal to bet amount
		case dealerScore > 21:
			playerResult = " 💰 ¡GANADOR!"
			netResult = payout - bet // Win amount minus original bet
		case blackjack.CompareHands(hand.Cards, game.Dealer.Cards) > 0:
			// Player wins (including blackjack over non-blackjack 21)
			if blackjack.IsBlackjack(hand.Cards) {
				playerResult = " 💰 ¡BLACKJACK! ¡GANADOR!"
				// For blackjack, payout is already correct (includes original bet + 3:2 winnings)
				netResult = payout - bet
			} else {
				playerResult = " 💰 ¡GANADOR!"
				netResult = payout - bet // Win amount minus original bet
			}
		case blackjack.CompareHands(hand.Cards, game.Dealer.Cards) < 0:
			// Dealer wins
			playerResult = " 🤦‍♂️ ¡PERDEDOR!"
			netResult = -bet // Loss equal to bet amount
		case blackjack.CompareHands(hand.Cards, game.Dealer.Cards) == 0:
			// Push (tie)
			if blackjack.IsBlackjack(hand.Cards) && blackjack.IsBlackjack(game.Dealer.Cards) {
				playerResult = ":beers: ¡EMPATE con BLACKJACK!"
			} else {
				playerResult = ":beers: ¡EMPATE!"
			}
			netResult = 0 // No gain or loss on a push (original bet is returned)
		}

		// Get member info for display name
		member, err := s.GuildMember(guildID, playerID)
		playerName := "Unknown Player"
		if err == nil && member.Nick != "" {
			playerName = member.Nick
		} else if err == nil && member.User != nil {
			playerName = member.User.Username
		}

		// Format the net result with color
		var netResultStr string
		if netResult > 0 {
			// Green for winnings
			netResultStr = fmt.Sprintf(" **+$%d**", netResult)
		} else if netResult < 0 {
			// Red for losses
			if hand.IsDoubledDown() {
				// For doubled down hands, show the actual bet amount for both bets
				// In a double down, both the original bet and double down bet are the same amount
				doubleDownBet := hand.GetDoubleDownBet()
				netResultStr = fmt.Sprintf(" **Bet: -$%d, Double Down: -$%d**", doubleDownBet, doubleDownBet)
			} else {
				// Regular loss
				netResultStr = fmt.Sprintf(" **-$%d**", -netResult)
			}
		} else {
			// Gray for push/tie
			if hand.IsDoubledDown() {
				// For doubled down hands that push, show that both bets were returned
				doubleDownBet := hand.GetDoubleDownBet()
				netResultStr = fmt.Sprintf(" **Bet: ±$%d, Double Down: ±$%d**", doubleDownBet, doubleDownBet)
			} else {
				// Regular push
				netResultStr = " **±$0**"
			}
		}

		results += fmt.Sprintf("**%s**: %s (%d)%s\n", playerName, playerResult, playerScore, netResultStr)
	}

	return results
}

// createGameButtons creates the action buttons if the game is in progress
func createGameButtons(game *blackjack.Game) []discordgo.MessageComponent {
	switch game.State {
	case blackjack.StateSplitting:
		// Get the current player whose turn it is to split
		currentPlayerID := game.GetCurrentSplittingPlayerID()
		if currentPlayerID == "" {
			// No current player, just show Play Again button
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

		// Show split options for the current player
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Split",
						Style:    discordgo.PrimaryButton,
						CustomID: "split",
					},
					discordgo.Button{
						Label:    "Skip Split",
						Style:    discordgo.SecondaryButton,
						CustomID: "decline_split",
					},
				},
			},
		}

	case blackjack.StateSpecialBets:
		// Get the current player whose turn it is for special bets
		currentPlayerID, err := game.GetCurrentSpecialBetsPlayerID()
		if err != nil || currentPlayerID == "" {
			// No current player or error, just show Play Again button
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

		// Create buttons for special bets based on eligibility
		components := []discordgo.MessageComponent{}
		actionRow := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{},
		}

		// Add Double Down button if eligible
		if game.IsEligibleForDoubleDown(currentPlayerID) {
			actionRow.Components = append(actionRow.Components, discordgo.Button{
				Label:    "Double Down (Hit)",
				Style:    discordgo.PrimaryButton,
				CustomID: "double_down",
			})
		}

		// Add Insurance button if eligible
		if game.IsEligibleForInsurance() {
			actionRow.Components = append(actionRow.Components, discordgo.Button{
				Label:    "Insurance",
				Style:    discordgo.PrimaryButton,
				CustomID: "insurance",
			})
		}

		// Always add Skip button
		actionRow.Components = append(actionRow.Components, discordgo.Button{
			Label:    "Skip Special Bets",
			Style:    discordgo.SecondaryButton,
			CustomID: "decline_special",
		})

		// Only add the action row if it has components
		if len(actionRow.Components) > 0 {
			components = append(components, actionRow)
		}

		return components

	case entities.StatePlaying:
		// Get the current player's ID
		var currentPlayerID string
		if len(game.PlayerOrder) > 0 && game.CurrentTurn < len(game.PlayerOrder) {
			currentPlayerID = game.PlayerOrder[game.CurrentTurn]
		}

		// Check if the current player has doubled down
		showHitButton := true
		if currentPlayerID != "" {
			hand, exists := game.Players[currentPlayerID]
			if exists && hand.IsDoubledDown() {
				// If player has doubled down, don't show hit button
				showHitButton = false
			}
		}

		// Create components based on whether hit button should be shown
		components := []discordgo.MessageComponent{}
		actionsRow := discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{},
		}

		// Only add hit button if player hasn't doubled down
		if showHitButton {
			actionsRow.Components = append(actionsRow.Components, discordgo.Button{
				Label:    "Hit",
				Style:    discordgo.PrimaryButton,
				CustomID: "hit",
			})
		}

		// Always show stand button
		actionsRow.Components = append(actionsRow.Components, discordgo.Button{
			Label:    "Stand",
			Style:    discordgo.SecondaryButton,
			CustomID: "stand",
		})

		components = append(components, actionsRow)
		return components

	default:
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
}

// createDealerField creates the dealer's hand field
func createDealerField(game *blackjack.Game) *discordgo.MessageEmbedField {
	// If we're in WAITING or BETTING state, dealer has no cards yet
	if game.State == entities.StateWaiting || game.State == entities.StateBetting || len(game.Dealer.Cards) == 0 {
		return &discordgo.MessageEmbedField{
			Name:   "🎩 El Dealer (Tuco)",
			Value:  "*Tuco shuffles the deck, waiting for bets...*",
			Inline: true,
		}
	}

	dealerScore := blackjack.GetBestScore(game.Dealer.Cards)
	dealerStatus := getStatusMessage(game.Dealer.Status)

	var dealerValue string
	if game.State == entities.StateComplete || game.State == entities.StateDealer {
		// Show all cards at the end of the game or during dealer's turn
		dealerValue = fmt.Sprintf("%s\nScore: %d%s", FormatCards(game.Dealer.Cards), dealerScore, dealerStatus)
	} else if game.State == entities.StateDealing {
		// During dealing, show cards being dealt with animation
		dealerValue = fmt.Sprintf("%s\n*Tuco deals the cards with a flourish*", FormatCards(game.Dealer.Cards))
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

// getRandomDoubleDownMessage returns a random Tuco-flavored message for double down actions
func getRandomDoubleDownMessage(playerName string) string {
	messages := []string{
		"¡Orale! %s is doubling down! One card and you're done, ese. That's how we roll at Tuco's table!",
		"Look at %s going all in with a double down! Remember, you get one card only - house rules, homes!",
		"¡Ay caramba! %s is doubling down! One more card coming your way, then you stand tight like a statue, comprende?",
		"Tuco sees %s has some serious cojones doubling down! One card only, then you're locked in like my abuela's secret recipe!",
		"¡Dios mío! %s is doubling their bet! You get one hit - just one - then you stand like a good hombre!",
		"¡Mira, mira! %s doubled down! Tuco will give you one card, then your fate is sealed like concrete shoes, amigo!",
		"¡Arriba, arriba! %s doubled down! One card coming your way, then you're standing tighter than Tuco's security at the back door!",
		"¡Qué valiente! %s is doubling down! Tuco gives you one card only - that's the rule at this establishment!",
	}

	// Pick a random message
	rand.Seed(time.Now().UnixNano())
	selectedMessage := messages[rand.Intn(len(messages))]

	return fmt.Sprintf(selectedMessage, playerName)
}
