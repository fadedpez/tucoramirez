package games

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot/cards"
)

// Interface for Discord session operations needed by the blackjack game
type BlackjackSession interface {
	InteractionRespond(i *discordgo.Interaction, r *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageEdit(channelID string, messageID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

const (
	Waiting  = "waiting"
	Playing  = "playing"
	Finished = "finished"

	maxPlayers = 6
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
	Deck          *cards.Deck
	Players       []*Player
	PlayerHand    []cards.Card
	DealerHand    []cards.Card
	GameState     string
	CreatorID     string
	ChannelID     string
	InteractionID string
	MessageID     string
}

var activeGames = make(map[string]*BlackjackGame) // key is channelID

// StartBlackjackGame starts a new game of blackjack
func StartBlackjackGame(s BlackjackSession, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID

	// Check if there's already a game in this channel
	if game, exists := activeGames[channelID]; exists {
		if game.GameState == Finished {
			// Clean up finished game
			delete(activeGames, channelID)
		} else {
			sendEphemeralError(s, i, "A game is already in progress in this channel!")
			return
		}
	}

	// Create and shuffle new deck
	deck, err := cards.NewDeck()
	if err != nil {
		sendEphemeralError(s, i, "Failed to create deck")
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
		Deck:          deck,
		Players:       []*Player{creator},
		GameState:     Waiting,
		CreatorID:     creator.ID,
		ChannelID:     channelID,
		InteractionID: i.ID,
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
	msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content:    fmt.Sprintf("🎲 Blackjack Game (1/%d players)\nPlayers: %s", maxPlayers, creator.Username),
		Components: buttons,
	})
	if err != nil {
		fmt.Printf("Error sending game start message: %v\n", err)
		return
	}

	// Store the message ID
	game.MessageID = msg.ID
}

// HandleBlackjackButton handles button presses for blackjack games
func HandleBlackjackButton(s BlackjackSession, i *discordgo.InteractionCreate) {
	// Get the game for this channel
	game, exists := activeGames[i.ChannelID]
	if !exists {
		sendEphemeralError(s, i, "No game in progress in this channel!")
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
		sendEphemeralError(s, i, "Unknown button interaction")
	}
}

// sendEphemeralError sends an ephemeral error message to the user
func sendEphemeralError(s BlackjackSession, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ %s", msg),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		fmt.Printf("Error sending error message: %v\n", err)
	}
}

// handleJoin handles a player joining the game
func handleJoin(s BlackjackSession, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if game.GameState != Waiting {
		sendEphemeralError(s, i, "Game is not accepting new players!")
		return
	}

	// Check if player is already in the game
	for _, p := range game.Players {
		if p.ID == i.Member.User.ID {
			sendEphemeralError(s, i, "You are already in this game!")
			return
		}
	}

	if len(game.Players) >= maxPlayers {
		sendEphemeralError(s, i, "Game is full!")
		return
	}

	// Add the new player
	newPlayer := &Player{
		ID:       i.Member.User.ID,
		Username: i.Member.User.Username,
		Hand:     []cards.Card{},
	}
	game.Players = append(game.Players, newPlayer)

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

	// Update the game message
	content := fmt.Sprintf("🎲 Blackjack Game (%d/%d players)\nPlayers: %s", len(game.Players), maxPlayers, playerList.String())
	buttons := []discordgo.MessageComponent{
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
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: buttons,
		},
	})
	if err != nil {
		fmt.Printf("Error updating game message: %v\n", err)
	}
}

// startGame starts the actual game after initialization
func startGame(s BlackjackSession, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if game.GameState != Waiting {
		if i != nil {
			sendEphemeralError(s, i, "Game has already started!")
		}
		return
	}

	// If this is a manual start (not auto-start), check if the player is in the game
	if i != nil {
		playerInGame := false
		for _, p := range game.Players {
			if p.ID == i.Member.User.ID {
				playerInGame = true
				break
			}
		}
		if !playerInGame {
			sendEphemeralError(s, i, "You must join the game before you can start it!")
			return
		}
	}

	if len(game.Players) < 1 {
		if i != nil {
			sendEphemeralError(s, i, "Not enough players to start!")
		}
		return
	}

	// If this is a manual start, check if the player is the creator
	if i != nil && i.Member.User.ID != game.CreatorID {
		sendEphemeralError(s, i, "Only the game creator can start the game!")
		return
	}

	// Set game state to playing
	game.GameState = Playing

	// Deal initial cards
	for _, player := range game.Players {
		// Draw two cards for the player
		card1, err := game.Deck.Draw()
		if err != nil {
			if i != nil {
				sendEphemeralError(s, i, "Failed to draw card")
			}
			return
		}
		card2, err := game.Deck.Draw()
		if err != nil {
			if i != nil {
				sendEphemeralError(s, i, "Failed to draw card")
			}
			return
		}
		player.Hand = []cards.Card{card1, card2}
		player.Score = calculateScore(player.Hand)
	}

	// Draw dealer's card
	dealerCard, err := game.Deck.Draw()
	if err != nil {
		if i != nil {
			sendEphemeralError(s, i, "Failed to draw dealer card")
		}
		return
	}
	game.DealerHand = []cards.Card{dealerCard}

	// Create game state message
	content := formatGameState(game)

	// Create action buttons
	buttons := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Hit",
					Style:    discordgo.SuccessButton,
					CustomID: "blackjack_hit",
				},
				discordgo.Button{
					Label:    "Stand",
					Style:    discordgo.DangerButton,
					CustomID: "blackjack_stand",
				},
			},
		},
	}

	// Send game state
	var err2 error
	if i != nil {
		err2 = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    content,
				Components: buttons,
			},
		})
	} else {
		// For manual start, first update the setup message to remove buttons
		_, err2 = s.ChannelMessageEdit(game.ChannelID, game.MessageID, fmt.Sprintf("🎲 Game Started! Players: %s", formatPlayerList(game.Players)))
		if err2 != nil {
			fmt.Printf("Error updating setup message: %v\n", err2)
		}

		// Then send the game state as a new message
		_, err2 = s.ChannelMessageSendComplex(game.ChannelID, &discordgo.MessageSend{
			Content:    content,
			Components: buttons,
		})
	}
	if err2 != nil {
		fmt.Printf("Error updating game state: %v\n", err2)
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

// handleHit handles the hit action
func handleHit(s BlackjackSession, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if game.GameState != Playing {
		sendEphemeralError(s, i, "Game is not in progress!")
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
		sendEphemeralError(s, i, "You are not in this game!")
		return
	}

	if currentPlayer.Stood || currentPlayer.Busted {
		sendEphemeralError(s, i, "You cannot hit - you have already stood or busted!")
		return
	}

	// Draw a card
	card, err := game.Deck.Draw()
	if err != nil {
		sendEphemeralError(s, i, "Failed to draw card")
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
						Style:    discordgo.SuccessButton,
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
		}
	}

	// If game is finished, add the winner message to the content
	if game.GameState == Finished {
		content = fmt.Sprintf("%s\n\n%s", content, determineWinner(game))
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

	// Clean up finished game
	if game.GameState == Finished {
		delete(activeGames, game.ChannelID)
	}
}

// handleStand handles the stand action
func handleStand(s BlackjackSession, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if game.GameState != Playing {
		sendEphemeralError(s, i, "Game is not in progress!")
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
		sendEphemeralError(s, i, "You are not in this game!")
		return
	}

	if currentPlayer.Stood || currentPlayer.Busted {
		sendEphemeralError(s, i, "You have already stood or busted!")
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
						Style:    discordgo.SuccessButton,
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
		}
	}

	// If game is finished, add the winner message to the content
	if game.GameState == Finished {
		content = fmt.Sprintf("%s\n\n%s", content, determineWinner(game))
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

// formatPlayerList formats a list of players into a string
func formatPlayerList(players []*Player) string {
	var playerList strings.Builder
	for i, player := range players {
		if i == len(players)-1 && len(players) > 1 {
			playerList.WriteString(", and ")
		}
		playerList.WriteString(player.Username)
		if i < len(players)-2 {
			playerList.WriteString(", ")
		}
	}
	return playerList.String()
}
