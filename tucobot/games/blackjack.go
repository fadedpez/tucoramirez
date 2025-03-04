package games

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/tucobot/cards"
)

var (
	gameSession *cards.GameSession
	cardValues = map[string]int{
		":two:": 2, ":three:": 3, ":four:": 4, ":five:": 5,
		":six:": 6, ":seven:": 7, ":eight:": 8, ":nine:": 9,
		":keycap_ten:": 10, ":regional_indicator_j:": 10,
		":regional_indicator_q:": 10, ":regional_indicator_k:": 10,
		":regional_indicator_a:": 11,
	}
)

func StartBlackjackGame(s *discordgo.Session, channelID string) {
	gameSession = NewGame()
	message := &discordgo.MessageSend{
		Content: "A new game of Blackjack has started! Use the buttons below to play.",
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
						Style:    discordgo.PrimaryButton,
						Disabled: true,
					},
					discordgo.Button{
						Label:    "Hit",
						CustomID: "hit_button",
						Style:    discordgo.PrimaryButton,
						Disabled: true,
					},
					discordgo.Button{
						Label:    "Stand",
						CustomID: "stand_button",
						Style:    discordgo.PrimaryButton,
						Disabled: true,
					},
				},
			},
		},
	}
	s.ChannelMessageSendComplex(channelID, message)
}

func NewGame() *cards.GameSession {
	deck := cards.CreateDeck()
	cards.ShuffleDeck(deck)

	dealer := &cards.Player{
		ID:    "dealer",
		Name:  "Dealer",
		Hand:  []cards.Card{},
		Score: 0,
	}

	return &cards.GameSession{
		Deck:    deck,
		Players: []cards.Player{},
		Dealer:  dealer,
	}
}

func HandleButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.MessageComponentData().CustomID {
	case "join_button":
		handleJoin(s, i)
	case "deal_button":
		handleDeal(s, i)
	case "hit_button":
		handleHit(s, i)
	case "stand_button":
		handleStand(s, i)
	}
}

func handleJoin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if gameSession == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "No active game. Please start a new game.",
			},
		})
		return
	}

	// Check if player already joined
	for _, p := range gameSession.Players {
		if p.ID == i.Member.User.ID {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You've already joined the game!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
	}

	// Add new player
	player := cards.Player{
		ID:    i.Member.User.ID,
		Name:  i.Member.User.Username,
		Hand:  []cards.Card{},
		Score: 0,
	}
	gameSession.Players = append(gameSession.Players, player)

	// Enable deal button if we have at least one player
	components := i.Message.Components
	if len(gameSession.Players) > 0 {
		components[0].(*discordgo.ActionsRow).Components[1].(*discordgo.Button).Disabled = false
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Current players: %s\nClick Deal to start!", getPlayerNames()),
			Components: components,
		},
	})
}

func handleDeal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if len(gameSession.Players) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No players have joined yet!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Deal initial cards
	for i := 0; i < 2; i++ {
		for j := range gameSession.Players {
			dealCard(&gameSession.Players[j])
		}
		if i == 0 {
			dealCard(gameSession.Dealer)
		}
	}

	// Enable Hit and Stand buttons, disable Join and Deal
	components := i.Message.Components
	components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.Button).Disabled = true  // Join
	components[0].(*discordgo.ActionsRow).Components[1].(*discordgo.Button).Disabled = true  // Deal
	components[0].(*discordgo.ActionsRow).Components[2].(*discordgo.Button).Disabled = false // Hit
	components[0].(*discordgo.ActionsRow).Components[3].(*discordgo.Button).Disabled = false // Stand

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: getGameState(),
			Components: components,
		},
	})
}

func handleHit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	player := findPlayer(i.Member.User.ID)
	if player == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You're not in this game!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	dealCard(player)
	updatePlayerScore(player)

	if player.Score > 21 {
		handleBust(s, i, player)
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: getGameState(),
			Components: i.Message.Components,
		},
	})
}

func handleStand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	player := findPlayer(i.Member.User.ID)
	if player == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You're not in this game!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Dealer's turn
	for gameSession.Dealer.Score < 17 {
		dealCard(gameSession.Dealer)
		updatePlayerScore(gameSession.Dealer)
	}

	// Determine winner
	result := determineWinner(player)

	// Disable all buttons
	components := i.Message.Components
	for j := range components[0].(*discordgo.ActionsRow).Components {
		components[0].(*discordgo.ActionsRow).Components[j].(*discordgo.Button).Disabled = true
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s\n%s", getGameState(), result),
			Components: components,
		},
	})
}

func handleBust(s *discordgo.Session, i *discordgo.InteractionCreate, player *cards.Player) {
	components := i.Message.Components
	for j := range components[0].(*discordgo.ActionsRow).Components {
		components[0].(*discordgo.ActionsRow).Components[j].(*discordgo.Button).Disabled = true
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("%s\n%s busted! Dealer wins!", getGameState(), player.Name),
			Components: components,
		},
	})
}

func dealCard(player *cards.Player) {
	if len(gameSession.Deck) == 0 {
		gameSession.Deck = cards.CreateDeck()
		cards.ShuffleDeck(gameSession.Deck)
	}
	
	card := gameSession.Deck[0]
	gameSession.Deck = gameSession.Deck[1:]
	player.Hand = append(player.Hand, card)
	updatePlayerScore(player)
}

func updatePlayerScore(player *cards.Player) {
	score := 0
	aces := 0

	for _, card := range player.Hand {
		if card.Value == ":regional_indicator_a:" {
			aces++
		} else {
			score += cardValues[card.Value]
		}
	}

	// Add aces
	for i := 0; i < aces; i++ {
		if score+11 <= 21 {
			score += 11
		} else {
			score += 1
		}
	}

	player.Score = score
}

func findPlayer(id string) *cards.Player {
	for i := range gameSession.Players {
		if gameSession.Players[i].ID == id {
			return &gameSession.Players[i]
		}
	}
	return nil
}

func getPlayerNames() string {
	names := make([]string, len(gameSession.Players))
	for i, p := range gameSession.Players {
		names[i] = p.Name
	}
	return strings.Join(names, ", ")
}

func getGameState() string {
	var sb strings.Builder
	
	// Show dealer's hand
	sb.WriteString("Dealer's hand: ")
	for i, card := range gameSession.Dealer.Hand {
		if i == 0 {
			sb.WriteString(fmt.Sprintf("%s%s ", card.Value, card.Suit))
		} else {
			sb.WriteString("🂠 ") // Hide dealer's hole card
		}
	}
	sb.WriteString(fmt.Sprintf("\nDealer's visible score: %d\n\n", cardValues[gameSession.Dealer.Hand[0].Value]))

	// Show each player's hand
	for _, p := range gameSession.Players {
		sb.WriteString(fmt.Sprintf("%s's hand: ", p.Name))
		for _, card := range p.Hand {
			sb.WriteString(fmt.Sprintf("%s%s ", card.Value, card.Suit))
		}
		sb.WriteString(fmt.Sprintf("\nScore: %d\n\n", p.Score))
	}

	return sb.String()
}

func determineWinner(player *cards.Player) string {
	if player.Score > 21 {
		return fmt.Sprintf("%s busted! Dealer wins!", player.Name)
	}
	if gameSession.Dealer.Score > 21 {
		return fmt.Sprintf("Dealer busted! %s wins!", player.Name)
	}
	if player.Score > gameSession.Dealer.Score {
		return fmt.Sprintf("%s wins!", player.Name)
	}
	if player.Score < gameSession.Dealer.Score {
		return "Dealer wins!"
	}
	return "It's a tie!"
}
