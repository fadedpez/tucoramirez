package games

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot/cards"
)

// Interface for Discord session operations
type sessionHandler interface {
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

// Game state
type BlackjackGame struct {
	Deck        *cards.Deck
	PlayerHand  []cards.Card
	DealerHand  []cards.Card
	GameOver    bool
	PlayerStood bool
}

var activeGames = make(map[string]*BlackjackGame)

func StartBlackjackGame(s sessionHandler, i *discordgo.InteractionCreate) {
	// Create a new game
	deck, err := cards.NewDeck()
	if err != nil {
		handleError(s, i, "Failed to create deck", err)
		return
	}

	game := &BlackjackGame{
		Deck: deck,
	}
	if err := game.Deck.Shuffle(); err != nil {
		handleError(s, i, "Failed to shuffle deck", err)
		return
	}

	// Deal initial cards
	card, err := game.Deck.Draw()
	if err != nil {
		handleError(s, i, "Failed to draw card", err)
		return
	}
	game.PlayerHand = append(game.PlayerHand, card)

	card, err = game.Deck.Draw()
	if err != nil {
		handleError(s, i, "Failed to draw card", err)
		return
	}
	game.DealerHand = append(game.DealerHand, card)

	card, err = game.Deck.Draw()
	if err != nil {
		handleError(s, i, "Failed to draw card", err)
		return
	}
	game.PlayerHand = append(game.PlayerHand, card)

	card, err = game.Deck.Draw()
	if err != nil {
		handleError(s, i, "Failed to draw card", err)
		return
	}
	game.DealerHand = append(game.DealerHand, card)

	// Store the game
	activeGames[i.Member.User.ID] = game

	// Create buttons
	buttons := []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Hit",
			Style:    discordgo.PrimaryButton,
			CustomID: "blackjack_hit",
		},
		discordgo.Button{
			Label:    "Stand",
			Style:    discordgo.SecondaryButton,
			CustomID: "blackjack_stand",
		},
	}

	// Send initial game state
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: formatGameState(game, false),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: buttons},
			},
		},
	}); err != nil {
		handleError(s, i, "Failed to start game", err)
		delete(activeGames, i.Member.User.ID) // Clean up if we can't start
		return
	}
}

func HandleBlackjackButton(s sessionHandler, i *discordgo.InteractionCreate) {
	// Get the game
	game, exists := activeGames[i.Member.User.ID]
	if !exists {
		handleError(s, i, "No active game found. Start a new game with /blackjack", nil)
		return
	}

	// Handle the button press
	switch i.MessageComponentData().CustomID {
	case "blackjack_hit":
		handleHit(s, i, game)
	case "blackjack_stand":
		handleStand(s, i, game)
	}
}

func handleHit(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	// Draw a card
	card, err := game.Deck.Draw()
	if err != nil {
		handleError(s, i, "Failed to draw card", err)
		return
	}
	game.PlayerHand = append(game.PlayerHand, card)

	// Check for bust
	if calculateScore(game.PlayerHand) > 21 {
		game.GameOver = true
		delete(activeGames, i.Member.User.ID)
	}

	// Update the game state
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    formatGameState(game, game.GameOver),
			Components: getButtons(game.GameOver),
		},
	}); err != nil {
		handleError(s, i, "Failed to update game state", err)
		return
	}
}

func handleStand(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	game.PlayerStood = true

	// Dealer draws until 17 or higher
	for calculateScore(game.DealerHand) < 17 {
		card, err := game.Deck.Draw()
		if err != nil {
			handleError(s, i, "Failed to draw card", err)
			return
		}
		game.DealerHand = append(game.DealerHand, card)
	}

	game.GameOver = true
	delete(activeGames, i.Member.User.ID)

	// Update the game state
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    formatGameState(game, true),
			Components: getButtons(true),
		},
	}); err != nil {
		handleError(s, i, "Failed to update game state", err)
		return
	}
}

func handleError(s sessionHandler, i *discordgo.InteractionCreate, msg string, err error) {
	errMsg := msg
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", msg, err)
	}
	
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: errMsg,
			Components: []discordgo.MessageComponent{},
		},
	}); err != nil {
		fmt.Printf("Error sending error response: %v\n", err)
	}
}

func calculateScore(hand []cards.Card) int {
	score := 0
	aces := 0

	for _, card := range hand {
		if card.Rank == "A" {
			aces++
			score += 11
		} else {
			score += card.Value()
		}
	}

	// Adjust for aces
	for aces > 0 && score > 21 {
		score -= 10
		aces--
	}

	return score
}

func formatGameState(game *BlackjackGame, showAll bool) string {
	playerScore := calculateScore(game.PlayerHand)
	dealerScore := calculateScore(game.DealerHand)

	var sb strings.Builder
	sb.WriteString("🎲 Blackjack Game 🎲\n\n")

	// Dealer's hand
	sb.WriteString("Dealer's hand: ")
	if showAll {
		for _, card := range game.DealerHand {
			sb.WriteString(card.String() + " ")
		}
		sb.WriteString(fmt.Sprintf("(Score: %d)", dealerScore))
	} else {
		sb.WriteString(game.DealerHand[0].String() + " [Hidden Card]")
	}
	sb.WriteString("\n\n")

	// Player's hand
	sb.WriteString("Your hand: ")
	for _, card := range game.PlayerHand {
		sb.WriteString(card.String() + " ")
	}
	sb.WriteString(fmt.Sprintf("(Score: %d)", playerScore))
	sb.WriteString("\n\n")

	// Game result
	if game.GameOver {
		if playerScore > 21 {
			sb.WriteString("🚫 Bust! You lose! 🚫")
		} else if game.PlayerStood {
			if dealerScore > 21 {
				sb.WriteString("🎉 Dealer busts! You win! 🎉")
			} else if playerScore > dealerScore {
				sb.WriteString("🎉 You win! 🎉")
			} else if playerScore < dealerScore {
				sb.WriteString("😢 Dealer wins! 😢")
			} else {
				sb.WriteString("🤝 Push! It's a tie! 🤝")
			}
		}
	}

	return sb.String()
}

func getButtons(gameOver bool) []discordgo.MessageComponent {
	if gameOver {
		return nil
	}

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Hit",
					Style:    discordgo.PrimaryButton,
					CustomID: "blackjack_hit",
				},
				discordgo.Button{
					Label:    "Stand",
					Style:    discordgo.SecondaryButton,
					CustomID: "blackjack_stand",
				},
			},
		},
	}
}
