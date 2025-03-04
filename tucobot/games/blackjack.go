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
}

const (
	maxPlayers = 6
	joinTimeout = 30 * time.Second
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
	Deck        *cards.Deck
	Players     []*Player
	PlayerHand  []cards.Card
	DealerHand  []cards.Card
	GameState   string
	CreatorID   string
	ChannelID   string
	CreatedAt   time.Time
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
	}

	game := &BlackjackGame{
		Deck:      deck,
		Players:   []*Player{creator},
		GameState: Waiting,
		CreatorID: i.Member.User.ID,
		ChannelID: channelID,
		CreatedAt: time.Now(),
	}
	activeGames[channelID] = game

	// Create the initial game message with Join and Start buttons
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🎲 Blackjack Game Created! 🎲\nPlayers (%d/%d):\n- %s\n\nClick Join to join the game, or Start to begin with current players.", len(game.Players), maxPlayers, creator.Username),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Join Game",
							Style:    discordgo.SuccessButton,
							CustomID: "blackjack_join",
						},
						discordgo.Button{
							Label:    "Start Game",
							Style:    discordgo.PrimaryButton,
							CustomID: "blackjack_start",
						},
					},
				},
			},
		},
	})

	if err != nil {
		fmt.Printf("Error sending game creation message: %v\n", err)
		return
	}

	// Start auto-start timer
	go func() {
		time.Sleep(autoStartTimeout)

		// Check if game still exists and is in waiting state
		game, exists := activeGames[channelID]
		if !exists || game.GameState != Waiting {
			return
		}

		// Send auto-start message
		_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: "30 seconds have passed. Starting game automatically!",
		})
		if err != nil {
			fmt.Printf("Error sending auto-start message: %v\n", err)
			return
		}

		// Start the game
		game.GameState = Playing
		
		// Draw initial cards for each player
		for _, player := range game.Players {
			// Draw two cards for the player
			card1, err := game.Deck.Draw()
			if err != nil {
				fmt.Printf("Error drawing card: %v\n", err)
				return
			}
			card2, err := game.Deck.Draw()
			if err != nil {
				fmt.Printf("Error drawing card: %v\n", err)
				return
			}
			player.Hand = []cards.Card{card1, card2}
			player.Score = calculateScore(player.Hand)
		}
		
		// Draw dealer's card
		dealerCard, err := game.Deck.Draw()
		if err != nil {
			fmt.Printf("Error drawing dealer card: %v\n", err)
			return
		}
		game.DealerHand = []cards.Card{dealerCard}

		// Send game state message
		msg := formatGameState(game)
		_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: msg,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Hit",
							Style:    discordgo.PrimaryButton,
							CustomID: "blackjack_hit",
						},
						discordgo.Button{
							Label:    "Stand",
							Style:    discordgo.DangerButton,
							CustomID: "blackjack_stand",
						},
					},
				},
			},
		})
		if err != nil {
			fmt.Printf("Error sending game state message: %v\n", err)
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
							Style:    discordgo.SuccessButton,
							CustomID: "blackjack_join",
							Disabled: len(game.Players) >= maxPlayers,
						},
						discordgo.Button{
							Label:    "Start Game",
							Style:    discordgo.PrimaryButton,
							CustomID: "blackjack_start",
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

	// Start the game
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
		handleError(s, i, "Failed to draw dealer's card", err)
		return
	}
	game.DealerHand = []cards.Card{dealerCard}

	// Update game display
	updateGameDisplay(s, i, game)
}

// updateGameDisplay updates the game display message
func updateGameDisplay(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	var content string
	var components []discordgo.MessageComponent

	switch game.GameState {
	case Playing:
		content = formatGameState(game)
		components = []discordgo.MessageComponent{
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
	case Finished:
		content = formatGameState(game) + "\n" + determineWinner(game)
		// No components needed for finished game
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
		},
	})
	if err != nil {
		handleError(s, i, "Error updating game display", err)
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

	// Find the player who clicked
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
			game.GameState = Finished
			playDealerHand(game)
		}
	}

	// Update game display
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: formatGameState(game),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Hit",
							Style:    discordgo.PrimaryButton,
							CustomID: "blackjack_hit",
							Disabled: game.GameState == Finished,
						},
						discordgo.Button{
							Label:    "Stand",
							Style:    discordgo.DangerButton,
							CustomID: "blackjack_stand",
							Disabled: game.GameState == Finished,
						},
					},
				},
			},
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

	// Find the player who clicked
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
		game.GameState = Finished
		playDealerHand(game)
	}

	// Update game display
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: formatGameState(game),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Hit",
							Style:    discordgo.PrimaryButton,
							CustomID: "blackjack_hit",
							Disabled: game.GameState == Finished,
						},
						discordgo.Button{
							Label:    "Stand",
							Style:    discordgo.DangerButton,
							CustomID: "blackjack_stand",
							Disabled: game.GameState == Finished,
						},
					},
				},
			},
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
