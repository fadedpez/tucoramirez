package blackjack

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/utils"
	"github.com/fadedpez/tucoramirez/pkg/cards"
	"github.com/fadedpez/tucoramirez/pkg/games/common"
)

const (
	MaxPlayers = 6
	StateWaiting = "waiting"
	StatePlaying = "playing"
	StateFinished = "finished"
)

// Game represents a single game of blackjack
type Game struct {
	Deck          *cards.Deck
	Players       []*common.Player
	DealerHand    []cards.Card
	State         string
	CreatorID     string
	ChannelID     string
	MessageID     string
	CurrentPlayer int
	mu            sync.RWMutex
}

// NewGame creates a new blackjack game
func NewGame(creatorID, channelID string) *Game {
	deck, _ := cards.NewDeck()
	deck.Shuffle()

	return &Game{
		Deck:      deck,
		Players:   make([]*common.Player, 0, MaxPlayers),
		State:     StateWaiting,
		CreatorID: creatorID,
		ChannelID: channelID,
	}
}

// AddPlayer adds a player to the game
func (g *Game) AddPlayer(player *common.Player) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.Players) >= MaxPlayers {
		return utils.NewGameError(utils.ErrTooManyPlayers, "Maximum number of players reached")
	}

	g.Players = append(g.Players, player)
	return nil
}

// Start begins the game
func (g *Game) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.Players) < 1 {
		return utils.NewGameError(utils.ErrNotEnoughPlayers, "Not enough players to start")
	}

	// Deal initial cards
	g.dealInitialCards()
	g.State = StatePlaying
	g.CurrentPlayer = 0
	return nil
}

// dealInitialCards deals two cards to each player and the dealer
func (g *Game) dealInitialCards() {
	// Deal to players
	for i := 0; i < 2; i++ {
		for _, player := range g.Players {
			card, _ := g.Deck.Draw()
			player.AddCard(card)
		}
		// Deal to dealer
		card, _ := g.Deck.Draw()
		g.DealerHand = append(g.DealerHand, card)
	}

	// Calculate initial scores
	for _, player := range g.Players {
		player.SetScore(calculateScore(player.GetHand()))
	}
}

// HandleJoin handles a player joining the game
func (g *Game) HandleJoin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Check if user is already in the game
	for _, p := range g.Players {
		if p.ID == i.Member.User.ID {
			utils.SendErrorResponse(s, i, utils.NewGameError("ALREADY_JOINED", "You're already in the game!"))
			return
		}
	}

	// Add new player
	player := &common.Player{
		ID:       i.Member.User.ID,
		Username: i.Member.User.Username,
	}
	
	if err := g.AddPlayer(player); err != nil {
		utils.SendErrorResponse(s, i, err)
		return
	}

	// Update message with new player list
	content := fmt.Sprintf("🎲 Blackjack Game (%d/%d players)\nPlayers: %s",
		len(g.Players), MaxPlayers, formatPlayerList(g.Players))

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Components: getGameButtons(len(g.Players) >= MaxPlayers, false),
		},
	})
	if err != nil {
		fmt.Printf("Error updating message: %v\n", err)
	}
}

// HandleStart handles starting the game
func (g *Game) HandleStart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if i.Member.User.ID != g.CreatorID {
		utils.SendErrorResponse(s, i, utils.NewGameError("NOT_CREATOR", "Only the game creator can start the game"))
		return
	}

	if err := g.Start(); err != nil {
		utils.SendErrorResponse(s, i, err)
		return
	}

	// Update message with game state
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: g.formatGameState(),
			Components: getPlayButtons(false),
		},
	})
	if err != nil {
		fmt.Printf("Error updating message: %v\n", err)
	}
}

// HandleHit handles a player hitting
func (g *Game) HandleHit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.validatePlayerTurn(i.Member.User.ID); err != nil {
		utils.SendErrorResponse(s, i, err)
		return
	}

	player := g.Players[g.CurrentPlayer]
	card, _ := g.Deck.Draw()
	player.AddCard(card)
	player.SetScore(calculateScore(player.GetHand()))

	if player.GetScore() > 21 {
		player.Bust()
		g.nextPlayer()
	}

	// Update message with new game state
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content: g.formatGameState(),
			Components: getPlayButtons(player.HasBusted()),
		},
	})
	if err != nil {
		fmt.Printf("Error updating message: %v\n", err)
	}
}

// HandleStand handles a player standing
func (g *Game) HandleStand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.validatePlayerTurn(i.Member.User.ID); err != nil {
		utils.SendErrorResponse(s, i, err)
		return
	}

	player := g.Players[g.CurrentPlayer]
	player.Stand()
	g.nextPlayer()

	var content string
	var components []discordgo.MessageComponent

	if g.IsFinished() {
		content = g.determineWinner()
		components = nil
	} else {
		content = g.formatGameState()
		components = getPlayButtons(false)
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    content,
			Components: components,
		},
	})
	if err != nil {
		fmt.Printf("Error updating message: %v\n", err)
	}
}

// nextPlayer moves to the next player or finishes the game
func (g *Game) nextPlayer() {
	g.CurrentPlayer++
	if g.CurrentPlayer >= len(g.Players) {
		g.finishGame()
	}
}

// finishGame completes the game and determines the winner
func (g *Game) finishGame() {
	g.playDealerHand()
	g.State = StateFinished
}

// playDealerHand plays out the dealer's hand
func (g *Game) playDealerHand() {
	dealerScore := calculateScore(g.DealerHand)
	for dealerScore < 17 {
		card, _ := g.Deck.Draw()
		g.DealerHand = append(g.DealerHand, card)
		dealerScore = calculateScore(g.DealerHand)
	}
}

// IsFinished returns whether the game is finished
func (g *Game) IsFinished() bool {
	return g.State == StateFinished
}

// SetMessageID sets the message ID for the game
func (g *Game) SetMessageID(id string) {
	g.MessageID = id
}

// validatePlayerTurn checks if it's the player's turn
func (g *Game) validatePlayerTurn(playerID string) error {
	if g.State != StatePlaying {
		return utils.NewGameError(utils.ErrGameAlreadyEnded, "The game has already ended")
	}
	if g.CurrentPlayer >= len(g.Players) || g.Players[g.CurrentPlayer].ID != playerID {
		return utils.NewGameError(utils.ErrNotPlayerTurn, "It's not your turn")
	}
	return nil
}

// Helper functions

func calculateScore(hand []cards.Card) int {
	score := 0
	aces := 0

	for _, card := range hand {
		if card.Rank == "A" {
			aces++
		} else if card.Rank == "K" || card.Rank == "Q" || card.Rank == "J" {
			score += 10
		} else {
			value := 0
			fmt.Sscanf(card.Rank, "%d", &value)
			score += value
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

	return score
}

func formatPlayerList(players []*common.Player) string {
	names := make([]string, len(players))
	for i, p := range players {
		names[i] = p.Username
	}
	return strings.Join(names, ", ")
}

func (g *Game) formatGameState() string {
	var sb strings.Builder

	// Show dealer's hand
	sb.WriteString("Dealer: ")
	if g.State == StatePlaying {
		// Show only first card during play
		sb.WriteString(g.DealerHand[0].String())
		sb.WriteString(" ??\n")
	} else {
		// Show full hand at end
		for i, card := range g.DealerHand {
			if i > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(card.String())
		}
		sb.WriteString(fmt.Sprintf(" (Score: %d)\n", calculateScore(g.DealerHand)))
	}

	// Show each player's hand
	for i, p := range g.Players {
		sb.WriteString(fmt.Sprintf("\n%s: ", p.Username))
		for j, card := range p.GetHand() {
			if j > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(card.String())
		}
		sb.WriteString(fmt.Sprintf(" (Score: %d)", p.GetScore()))
		if p.HasBusted() {
			sb.WriteString(" BUST!")
		} else if p.HasStood() {
			sb.WriteString(" STAND")
		} else if i == g.CurrentPlayer {
			sb.WriteString(" 👈 Your turn!")
		}
	}

	return sb.String()
}

func (g *Game) determineWinner() string {
	dealerScore := calculateScore(g.DealerHand)
	dealerBusted := dealerScore > 21

	var sb strings.Builder
	sb.WriteString(g.formatGameState())
	sb.WriteString("\n\n🏆 Results:\n")

	for _, p := range g.Players {
		playerScore := p.GetScore()
		if p.HasBusted() {
			sb.WriteString(fmt.Sprintf("%s: Lost (Bust)\n", p.Username))
		} else if dealerBusted {
			sb.WriteString(fmt.Sprintf("%s: Won! (Dealer bust)\n", p.Username))
		} else if playerScore > dealerScore {
			sb.WriteString(fmt.Sprintf("%s: Won! (%d > %d)\n", p.Username, playerScore, dealerScore))
		} else if playerScore < dealerScore {
			sb.WriteString(fmt.Sprintf("%s: Lost (%d < %d)\n", p.Username, playerScore, dealerScore))
		} else {
			sb.WriteString(fmt.Sprintf("%s: Push (%d = %d)\n", p.Username, playerScore, dealerScore))
		}
	}

	return sb.String()
}

func getGameButtons(joinDisabled bool, startDisabled bool) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Join Game",
					Style:    discordgo.SuccessButton,
					CustomID: "blackjack_join",
					Disabled: joinDisabled,
				},
				discordgo.Button{
					Label:    "Start Game",
					Style:    discordgo.PrimaryButton,
					CustomID: "blackjack_start",
					Disabled: startDisabled,
				},
			},
		},
	}
}

func getPlayButtons(disabled bool) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Hit",
					Style:    discordgo.SuccessButton,
					CustomID: "blackjack_hit",
					Disabled: disabled,
				},
				discordgo.Button{
					Label:    "Stand",
					Style:    discordgo.DangerButton,
					CustomID: "blackjack_stand",
					Disabled: disabled,
				},
			},
		},
	}
}
