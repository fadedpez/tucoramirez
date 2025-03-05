package games

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot/cards"
)

// Interface for Discord session operations
type sessionHandler interface {
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error)
	InteractionResponseEdit(i *discordgo.Interaction, edit *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

const (
	maxPlayers       = 6
	joinTimeout      = 30 * time.Second
	autoStartTimeout = 30 * time.Second
)

// Game states
const (
	Waiting  = "WAITING"
	Playing  = "PLAYING"
	Finished = "FINISHED"
)

// Player represents a player in the game
type Player struct {
	ID       string
	Username string
	Hand     []cards.Card
	Score    int
	Stood    bool
	Busted   bool
}

// BlackjackGame represents a game of blackjack
type BlackjackGame struct {
	Deck       *cards.Deck
	Players    []*Player
	PlayerHand []cards.Card
	DealerHand []cards.Card
	GameState  string
	CreatorID  string
	ChannelID  string
	CreatedAt  time.Time
}

var activeGames = make(map[string]*BlackjackGame) // key is channelID

// StartBlackjackGame starts a new game of blackjack
func StartBlackjackGame(s sessionHandler, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID

	// Check if there's already a game in this channel
	if game, exists := activeGames[channelID]; exists {
		if game.GameState == Finished {
			// Clean up finished game
			delete(activeGames, channelID)
		} else {
			handleError(s, i, "A game is already in progress in this channel!", nil)
			return
		}
	}

	// Create and shuffle new deck
	deck, err := cards.NewDeck()
	if err != nil {
		handleError(s, i, "Failed to create deck", err)
		return
	}
	deck.Shuffle()

	// Create initial player
	creator := &Player{
		ID:       i.Member.User.ID,
		Username: i.Member.User.Username,
		Hand:     []cards.Card{},
	}

	// Create new game
	game := &BlackjackGame{
		Deck:      deck,
		Players:   []*Player{creator},
		GameState: Waiting,
		CreatorID: creator.ID,
		ChannelID: channelID,
		CreatedAt: time.Now(),
	}

	// Store game in active games
	activeGames[channelID] = game

	// Create buttons for joining and starting
	buttons := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Join Game",
					Style:    discordgo.SuccessButton, // Success (green)
					CustomID: "blackjack_join",
				},
				discordgo.Button{
					Label:    "Start Game",
					Style:    discordgo.PrimaryButton, // Primary (blue)
					CustomID: "blackjack_start",
				},
			},
		},
	}

	// Send initial message
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content:    "Starting a new game of Blackjack! Click Join Game to join, or Start Game to begin. Game will auto-start in 30 seconds.",
			Components: buttons,
		},
	})
	if err != nil {
		fmt.Printf("Error sending game start message: %v\n", err)
		return
	}

	// Start auto-start timer in a goroutine
	go func() {
		time.Sleep(autoStartTimeout)
		
		// Check if game still exists and is in waiting state
		if game, exists := activeGames[channelID]; exists && game.GameState == Waiting && len(game.Players) > 0 {
			startGame(s, i, game)
		}
	}()
}

// HandleBlackjackButton handles button interactions for blackjack
func HandleBlackjackButton(s sessionHandler, i *discordgo.InteractionCreate) {
	// Get the game for this channel
	game, exists := activeGames[i.ChannelID]
	if !exists {
		handleError(s, i, "No game in progress in this channel!", nil)
		return
	}

	switch i.MessageComponentData().CustomID {
	case "blackjack_join":
		handleJoin(s, i, game)
	case "blackjack_start":
		startGame(s, i, game)
	case "blackjack_hit":
		handleHit(s, i, game)
	case "blackjack_stand":
		handleStand(s, i, game)
	default:
		handleError(s, i, "Unknown button interaction", nil)
	}
}

// handleJoin handles a player joining the game
func handleJoin(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if game.GameState != Waiting {
		handleError(s, i, "Game has already started!", nil)
		return
	}

	// Check if player is already in the game
	playerID := i.Member.User.ID
	for _, p := range game.Players {
		if p.ID == playerID {
			handleError(s, i, "You are already in the game!", nil)
			return
		}
	}

	// Check if game is full
	if len(game.Players) >= maxPlayers {
		handleError(s, i, "Game is full!", nil)
		return
	}

	// Add the new player
	newPlayer := &Player{
		ID:       playerID,
		Username: i.Member.User.Username,
	}
	game.Players = append(game.Players, newPlayer)

	// Update the message with current players
	var playerList strings.Builder
	for _, p := range game.Players {
		playerList.WriteString("- " + p.Username + "\n")
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🎲 Blackjack Game Created! 🎲\nPlayers (%d/%d):\n%s\nClick Join to join the game, or Start to begin with current players.",
				len(game.Players), maxPlayers, playerList.String()),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Join Game",
							Style:    discordgo.SuccessButton, // Success (green)
							CustomID: "blackjack_join",
							Disabled: len(game.Players) >= maxPlayers,
						},
						discordgo.Button{
							Label:    "Start Game",
							Style:    discordgo.PrimaryButton, // Primary (blue)
							CustomID: "blackjack_start",
							Disabled: len(game.Players) < 1,
						},
					},
				},
			},
		},
	})

	if err != nil {
		fmt.Printf("Error updating game message: %v\n", err)
	}
}

// startGame starts the actual game after initialization
func startGame(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if game.GameState != Waiting {
		handleError(s, i, "Game has already started!", nil)
		return
	}

	if len(game.Players) < 1 {
		handleError(s, i, "Not enough players to start!", nil)
		return
	}

	// Only creator can start the game
	if i.Member.User.ID != game.CreatorID {
		handleError(s, i, "Only the game creator can start the game!", nil)
		return
	}

	game.GameState = Playing

	// Draw initial cards for each player
	for _, player := range game.Players {
		// Draw two cards for the player
		card1, err := game.Deck.Draw()
		if err != nil {
			handleError(s, i, "Failed to draw card", err)
			return
		}
		card2, err := game.Deck.Draw()
		if err != nil {
			handleError(s, i, "Failed to draw card", err)
			return
		}
		player.Hand = []cards.Card{card1, card2}
		player.Score = calculateScore(player.Hand)
	}

	// Draw dealer's card
	dealerCard, err := game.Deck.Draw()
	if err != nil {
		handleError(s, i, "Failed to draw dealer card", err)
		return
	}
	game.DealerHand = []cards.Card{dealerCard}

	// Build the player list
	var playerList strings.Builder
	for i, player := range game.Players {
		if i == len(game.Players)-1 && len(game.Players) > 1 {
			playerList.WriteString(", and ")
		}
		playerList.WriteString(player.Username)
		if i < len(game.Players)-2 {
			playerList.WriteString(", ")
		}
	}

	// First, clear the join/start buttons by updating the original message
	content := fmt.Sprintf("🎲 Game Started! Players: %s 🎲", playerList.String())
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: []discordgo.MessageComponent{}, // Empty components removes all buttons
		},
	})
	if err != nil {
		fmt.Printf("Error clearing game setup message: %v\n", err)
		return
	}

	// Then send a new message with the game state and play buttons
	msg := formatGameState(game)
	_, err = s.ChannelMessageSendComplex(game.ChannelID, &discordgo.MessageSend{
		Content: msg,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Hit",
						Style:    discordgo.SuccessButton, // Success (green)
						CustomID: "blackjack_hit",
					},
					discordgo.Button{
						Label:    "Stand",
						Style:    discordgo.DangerButton, // Danger (red)
						CustomID: "blackjack_stand",
					},
				},
			},
		},
	})
	if err != nil {
		fmt.Printf("Error sending game state message: %v\n", err)
		return
	}
}

// formatGameState formats the game state message
func formatGameState(game *BlackjackGame) string {
	var sb strings.Builder
	sb.WriteString("🎲 **Blackjack Game** 🎲\n\n")

	// Show dealer's hand
	sb.WriteString("Dealer's hand: ")
	if game.GameState == Finished {
		for _, card := range game.DealerHand {
			sb.WriteString(card.String() + " ")
		}
		sb.WriteString(fmt.Sprintf("(Score: %d)", calculateScore(game.DealerHand)))
	} else {
		sb.WriteString(game.DealerHand[0].String() + " ?")
	}
	sb.WriteString("\n\n")

	// Show each player's hand
	for _, player := range game.Players {
		sb.WriteString(fmt.Sprintf("**%s's hand**: ", player.Username))
		for _, card := range player.Hand {
			sb.WriteString(card.String() + " ")
		}
		sb.WriteString(fmt.Sprintf("(Score: %d)", player.Score))
		if player.Busted {
			sb.WriteString(" 💥 BUST!")
		} else if player.Stood {
			sb.WriteString(" 🛑 STAND")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// determineWinner determines the winner of the game
func determineWinner(game *BlackjackGame) string {
	var sb strings.Builder
	sb.WriteString("🎲 **Game Summary** 🎲\n\n")

	// Show dealer's final hand
	dealerScore := calculateScore(game.DealerHand)
	sb.WriteString("**Dealer's hand**: ")
	for _, card := range game.DealerHand {
		sb.WriteString(card.String() + " ")
	}
	sb.WriteString(fmt.Sprintf("(Score: %d)", dealerScore))
	if dealerScore > 21 {
		sb.WriteString(" 💥 BUST!")
	}
	sb.WriteString("\n\n")

	// Show each player's result
	sb.WriteString("**Player Results**:\n")
	for _, player := range game.Players {
		sb.WriteString(fmt.Sprintf("• %s: ", player.Username))
		for _, card := range player.Hand {
			sb.WriteString(card.String() + " ")
		}
		sb.WriteString(fmt.Sprintf("(Score: %d)", player.Score))

		if player.Busted {
			sb.WriteString(" 💥 BUST!")
		} else if dealerScore > 21 {
			sb.WriteString(" 🎉 WIN!")
		} else if player.Score > dealerScore {
			sb.WriteString(" 🎉 WIN!")
		} else if player.Score == dealerScore {
			sb.WriteString(" 🤝 PUSH")
		} else {
			sb.WriteString(" 😢 LOSE")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// calculateScore calculates the score of a hand
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

// handleError handles error responses
func handleError(s sessionHandler, i *discordgo.InteractionCreate, msg string, err error) {
	errMsg := msg
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", msg, err)
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: errMsg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		fmt.Printf("Error sending error message: %v\n", err)
	}
}

// handleHit handles the hit action
func handleHit(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if game.GameState != Playing {
		handleError(s, i, "Game is not in progress!", nil)
		return
	}

	// Find the current player
	var currentPlayer *Player
	for _, p := range game.Players {
		if p.ID == i.Member.User.ID {
			currentPlayer = p
			break
		}
	}

	if currentPlayer == nil {
		handleError(s, i, "You are not in this game!", nil)
		return
	}

	if currentPlayer.Stood || currentPlayer.Busted {
		handleError(s, i, "You cannot hit - you have already stood or busted!", nil)
		return
	}

	// Draw a card
	card, err := game.Deck.Draw()
	if err != nil {
		handleError(s, i, "Failed to draw card", err)
		return
	}

	// Add card to player's hand
	currentPlayer.Hand = append(currentPlayer.Hand, card)
	currentPlayer.Score = calculateScore(currentPlayer.Hand)

	// Check for bust
	if currentPlayer.Score > 21 {
		currentPlayer.Busted = true

		// Check if all players are done
		allDone := true
		for _, p := range game.Players {
			if !p.Stood && !p.Busted {
				allDone = false
				break
			}
		}

		if allDone {
			// Play out dealer's hand
			playDealerHand(game)
			game.GameState = Finished
		}
	}

	// Update game display
	content := formatGameState(game)
	var components []discordgo.MessageComponent
	if game.GameState == Playing {
		components = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Hit",
						Style:    discordgo.SuccessButton, // Success (green)
						CustomID: "blackjack_hit",
						Disabled: game.GameState == Finished,
					},
					discordgo.Button{
						Label:    "Stand",
						Style:    discordgo.DangerButton, // Danger (red)
						CustomID: "blackjack_stand",
						Disabled: game.GameState == Finished,
					},
				},
			},
		}
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
		},
	})
	if err != nil {
		fmt.Printf("Error updating game display: %v\n", err)
		return
	}

	// If game is finished, show summary and clean up
	if game.GameState == Finished {
		_, err = s.ChannelMessageSendComplex(game.ChannelID, &discordgo.MessageSend{
			Content: determineWinner(game),
		})
		if err != nil {
			fmt.Printf("Error sending game summary: %v\n", err)
		}
		delete(activeGames, game.ChannelID)
	}
}

// handleStand handles the stand action
func handleStand(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if game.GameState != Playing {
		handleError(s, i, "Game is not in progress!", nil)
		return
	}

	// Find the current player
	var currentPlayer *Player
	for _, p := range game.Players {
		if p.ID == i.Member.User.ID {
			currentPlayer = p
			break
		}
	}

	if currentPlayer == nil {
		handleError(s, i, "You are not in this game!", nil)
		return
	}

	if currentPlayer.Stood || currentPlayer.Busted {
		handleError(s, i, "You have already stood or busted!", nil)
		return
	}

	// Mark player as stood
	currentPlayer.Stood = true

	// Check if all players are done
	allDone := true
	for _, p := range game.Players {
		if !p.Stood && !p.Busted {
			allDone = false
			break
		}
	}

	if allDone {
		// Play out dealer's hand
		playDealerHand(game)
		game.GameState = Finished
	}

	// Update game display
	content := formatGameState(game)
	var components []discordgo.MessageComponent
	if game.GameState == Playing {
		components = []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Hit",
						Style:    discordgo.SuccessButton, // Success (green)
						CustomID: "blackjack_hit",
						Disabled: game.GameState == Finished,
					},
					discordgo.Button{
						Label:    "Stand",
						Style:    discordgo.DangerButton, // Danger (red)
						CustomID: "blackjack_stand",
						Disabled: game.GameState == Finished,
					},
				},
			},
		}
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
		},
	})
	if err != nil {
		fmt.Printf("Error updating game display: %v\n", err)
		return
	}

	// If game is finished, show summary and clean up
	if game.GameState == Finished {
		_, err = s.ChannelMessageSendComplex(game.ChannelID, &discordgo.MessageSend{
			Content: determineWinner(game),
		})
		if err != nil {
			fmt.Printf("Error sending game summary: %v\n", err)
		}
		delete(activeGames, game.ChannelID)
	}
}

// playDealerHand plays out the dealer's hand
func playDealerHand(game *BlackjackGame) {
	// Dealer must hit on 16 and stand on 17
	for calculateScore(game.DealerHand) < 17 {
		card, err := game.Deck.Draw()
		if err != nil {
			fmt.Printf("Error drawing dealer card: %v\n", err)
			return
		}
		game.DealerHand = append(game.DealerHand, card)
	}
}
