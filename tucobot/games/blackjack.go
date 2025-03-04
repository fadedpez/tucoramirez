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
}

const (
	maxPlayers = 6
	joinTimeout = 30 * time.Second
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

// Game state
type BlackjackGame struct {
	Deck        *cards.Deck
	Players     []*Player
	DealerHand  []cards.Card
	GameStarted bool
	GameOver    bool
	CreatorID   string
	ChannelID   string
	CreatedAt   time.Time
}

var activeGames = make(map[string]*BlackjackGame) // key is channelID

func StartBlackjackGame(s sessionHandler, i *discordgo.InteractionCreate) {
	channelID := i.ChannelID

	// Check if there's already a game in this channel
	if _, exists := activeGames[channelID]; exists {
		handleError(s, i, "A game is already in progress in this channel!", nil)
		return
	}

	// Create a new deck
	deck, err := cards.NewDeck()
	if err != nil {
		handleError(s, i, "Failed to create deck", err)
		return
	}

	// Shuffle the deck
	if err := deck.Shuffle(); err != nil {
		handleError(s, i, "Failed to shuffle deck", err)
		return
	}
	
	game := &BlackjackGame{
		Deck:      deck,
		Players:   make([]*Player, 0),
		CreatorID: i.Member.User.ID,
		ChannelID: channelID,
		CreatedAt: time.Now(),
	}

	// Add the creator as the first player
	game.Players = append(game.Players, &Player{
		ID:       i.Member.User.ID,
		Username: i.Member.User.Username,
	})

	// Store the game
	activeGames[channelID] = game

	// Create buttons for joining and starting
	buttons := getJoinButtons()

	// Send initial game state
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: formatGameState(game, false),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: buttons,
				},
			},
		},
	}); err != nil {
		handleError(s, i, "Failed to start game", err)
		delete(activeGames, channelID)
		return
	}

	// Start a goroutine to handle game timeout
	go func() {
		time.Sleep(joinTimeout)
		if game, exists := activeGames[channelID]; exists && !game.GameStarted {
			delete(activeGames, channelID)
			s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
				Content: "Game has timed out due to inactivity.",
			})
		}
	}()
}

func HandleBlackjackButton(s sessionHandler, i *discordgo.InteractionCreate) {
	// Get the game
	game, exists := activeGames[i.ChannelID]
	if !exists {
		handleError(s, i, "No active game found in this channel. Start a new game with /blackjack", nil)
		return
	}

	// Handle the button press
	switch i.MessageComponentData().CustomID {
	case "blackjack_join":
		handleJoin(s, i, game)
	case "blackjack_start":
		handleStart(s, i, game)
	case "blackjack_hit":
		handleHit(s, i, game)
	case "blackjack_stand":
		handleStand(s, i, game)
	}
}

func handleJoin(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if game.GameStarted {
		handleError(s, i, "Game has already started!", nil)
		return
	}

	if len(game.Players) >= maxPlayers {
		handleError(s, i, "Game is full! Maximum 6 players.", nil)
		return
	}

	// Check if player is already in the game
	for _, p := range game.Players {
		if p.ID == i.Member.User.ID {
			// Send an ephemeral message instead of updating the game state
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You're already in this game!",
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				fmt.Printf("Error sending ephemeral message: %v\n", err)
			}
			return
		}
	}

	// Add the player
	game.Players = append(game.Players, &Player{
		ID:       i.Member.User.ID,
		Username: i.Member.User.Username,
	})

	// Update the game state
	buttons := getJoinButtons()
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: formatGameState(game, false),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: buttons,
				},
			},
		},
	}); err != nil {
		handleError(s, i, "Failed to update game state", err)
		return
	}
}

func handleStart(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if game.GameStarted {
		handleError(s, i, "Game has already started!", nil)
		return
	}

	if i.Member.User.ID != game.CreatorID {
		handleError(s, i, "Only the game creator can start the game!", nil)
		return
	}

	if len(game.Players) < 1 {
		handleError(s, i, "Need at least one player to start!", nil)
		return
	}

	game.GameStarted = true

	// Deal initial cards to all players
	for _, player := range game.Players {
		for j := 0; j < 2; j++ {
			card, err := game.Deck.Draw()
			if err != nil {
				handleError(s, i, "Failed to draw card", err)
				return
			}
			player.Hand = append(player.Hand, card)
		}
		player.Score = calculateScore(player.Hand)
	}

	// Deal dealer's cards
	for j := 0; j < 2; j++ {
		card, err := game.Deck.Draw()
		if err != nil {
			handleError(s, i, "Failed to draw card", err)
			return
		}
		game.DealerHand = append(game.DealerHand, card)
	}

	// Update the game state
	buttons := getGameButtons()
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: formatGameState(game, false),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: buttons,
				},
			},
		},
	}); err != nil {
		handleError(s, i, "Failed to update game state", err)
		return
	}
}

func handleHit(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if !game.GameStarted {
		handleError(s, i, "Game hasn't started yet!", nil)
		return
	}

	// Find the player
	var player *Player
	for _, p := range game.Players {
		if p.ID == i.Member.User.ID {
			player = p
			break
		}
	}

	if player == nil {
		handleError(s, i, "You're not in this game!", nil)
		return
	}

	if player.Stood || player.Busted {
		// Send an ephemeral message instead of updating game state
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Your turn is already over! %s", getPlayerStatusMessage(player)),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			fmt.Printf("Error sending ephemeral message: %v\n", err)
		}
		return
	}

	// Draw a card
	card, err := game.Deck.Draw()
	if err != nil {
		handleError(s, i, "Failed to draw card", err)
		return
	}

	player.Hand = append(player.Hand, card)
	player.Score = calculateScore(player.Hand)

	if player.Score > 21 {
		player.Busted = true
		// Check if game is over
		game.GameOver = checkGameOver(game)
	}

	// If game is over, reveal dealer's cards and play out dealer's hand
	if game.GameOver {
		playDealerHand(game)
		gameSummary := determineGameResults(game)
		
		// First update the interaction response without buttons
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: formatGameState(game, true),
				Components: []discordgo.MessageComponent{}, // Empty components removes the buttons
			},
		}); err != nil {
			handleError(s, i, "Failed to update game state", err)
			return
		}

		// Then send the final game summary as a new message
		if _, err := s.ChannelMessageSendComplex(game.ChannelID, &discordgo.MessageSend{
			Content: formatGameState(game, true) + gameSummary,
		}); err != nil {
			handleError(s, i, "Failed to send game summary", err)
			return
		}

		delete(activeGames, game.ChannelID)
		return
	}

	// Update the game state
	buttons := getGameButtons()
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: formatGameState(game, game.GameOver),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: buttons,
				},
			},
		},
	}); err != nil {
		handleError(s, i, "Failed to update game state", err)
		return
	}
}

func handleStand(s sessionHandler, i *discordgo.InteractionCreate, game *BlackjackGame) {
	if !game.GameStarted {
		handleError(s, i, "Game hasn't started yet!", nil)
		return
	}

	// Find the player
	var player *Player
	for _, p := range game.Players {
		if p.ID == i.Member.User.ID {
			player = p
			break
		}
	}

	if player == nil {
		handleError(s, i, "You're not in this game!", nil)
		return
	}

	if player.Stood || player.Busted {
		// Send an ephemeral message instead of updating game state
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Your turn is already over! %s", getPlayerStatusMessage(player)),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			fmt.Printf("Error sending ephemeral message: %v\n", err)
		}
		return
	}

	player.Stood = true

	// Check if game is over
	game.GameOver = checkGameOver(game)

	// If game is over, reveal dealer's cards and play out dealer's hand
	if game.GameOver {
		playDealerHand(game)
		gameSummary := determineGameResults(game)
		
		// First update the interaction response without buttons
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: formatGameState(game, true),
				Components: []discordgo.MessageComponent{}, // Empty components removes the buttons
			},
		}); err != nil {
			handleError(s, i, "Failed to update game state", err)
			return
		}

		// Then send the final game summary as a new message
		if _, err := s.ChannelMessageSendComplex(game.ChannelID, &discordgo.MessageSend{
			Content: formatGameState(game, true) + gameSummary,
		}); err != nil {
			handleError(s, i, "Failed to send game summary", err)
			return
		}

		delete(activeGames, game.ChannelID)
		return
	}

	// Update the game state with buttons for ongoing game
	buttons := getGameButtons()
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: formatGameState(game, game.GameOver),
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: buttons,
				},
			},
		},
	}); err != nil {
		handleError(s, i, "Failed to update game state", err)
		return
	}
}

func playDealerHand(game *BlackjackGame) {
	// Dealer draws until 17 or higher
	for calculateScore(game.DealerHand) < 17 {
		card, err := game.Deck.Draw()
		if err != nil {
			return // If we can't draw, just return
		}
		game.DealerHand = append(game.DealerHand, card)
	}
}

func checkGameOver(game *BlackjackGame) bool {
	allDone := true
	for _, p := range game.Players {
		if !p.Stood && !p.Busted {
			allDone = false
			break
		}
	}
	return allDone
}

func getJoinButtons() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Join Game",
			Style:    2, // Primary
			CustomID: "blackjack_join",
		},
		discordgo.Button{
			Label:    "Start Game",
			Style:    3, // Success
			CustomID: "blackjack_start",
		},
	}
}

func getGameButtons() []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.Button{
			Label:    "Hit",
			Style:    2, // Primary
			CustomID: "blackjack_hit",
		},
		discordgo.Button{
			Label:    "Stand",
			Style:    2, // Secondary
			CustomID: "blackjack_stand",
		},
	}
}

func formatGameState(game *BlackjackGame, showAll bool) string {
	var sb strings.Builder
	sb.WriteString("🎲 Blackjack Game 🎲\n\n")

	if !game.GameStarted {
		sb.WriteString(fmt.Sprintf("Players (%d/6):\n", len(game.Players)))
		for _, p := range game.Players {
			sb.WriteString(fmt.Sprintf("• %s\n", p.Username))
		}
		sb.WriteString("\nWaiting for more players to join...\n")
		return sb.String()
	}

	// Dealer's hand
	sb.WriteString("Dealer's hand: ")
	if showAll {
		for _, card := range game.DealerHand {
			sb.WriteString(card.String() + " ")
		}
		sb.WriteString(fmt.Sprintf("(Score: %d)", calculateScore(game.DealerHand)))
	} else {
		sb.WriteString(game.DealerHand[0].String() + " [Hidden Card]")
	}
	sb.WriteString("\n\n")

	// Players' hands
	for _, p := range game.Players {
		sb.WriteString(fmt.Sprintf("%s's hand: ", p.Username))
		for _, card := range p.Hand {
			sb.WriteString(card.String() + " ")
		}
		sb.WriteString(fmt.Sprintf("(Score: %d)", calculateScore(p.Hand)))
		if p.Busted {
			sb.WriteString(" 🚫 BUST!")
		} else if p.Stood {
			sb.WriteString(" 🛑 STAND")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

func determineGameResults(game *BlackjackGame) string {
	dealerScore := calculateScore(game.DealerHand)
	dealerBusted := dealerScore > 21

	var summary strings.Builder
	summary.WriteString("\n🎲 **Game Summary** 🎲\n")
	summary.WriteString(fmt.Sprintf("Dealer's final hand: %d\n", dealerScore))
	if dealerBusted {
		summary.WriteString("Dealer busted! 💥\n")
	}
	summary.WriteString("\nResults:\n")

	for _, p := range game.Players {
		playerScore := calculateScore(p.Hand)
		result := "Lost 😢"
		
		if p.Busted {
			result = "Busted 💥"
		} else if dealerBusted && !p.Busted {
			result = "Won! 🎉"
		} else if !p.Busted && playerScore > dealerScore {
			result = "Won! 🎉"
		} else if !p.Busted && playerScore == dealerScore {
			result = "Push 🤝"
		}
		
		summary.WriteString(fmt.Sprintf("%s: %d - %s\n", p.Username, playerScore, result))
	}
	
	return summary.String()
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

func handleError(s sessionHandler, i *discordgo.InteractionCreate, msg string, err error) {
	errMsg := msg
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", msg, err)
	}
	
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: errMsg,
			Components: []discordgo.MessageComponent{},
		},
	}); err != nil {
		fmt.Printf("Error sending error response: %v\n", err)
	}
}

func getPlayerStatusMessage(p *Player) string {
	if p.Busted {
		return "You busted! 💥"
	}
	if p.Stood {
		return "You stood! 🤚"
	}
	return ""
}
