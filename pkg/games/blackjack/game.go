package blackjack

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/fadedpez/tucoramirez/internal/discord"
	"github.com/fadedpez/tucoramirez/internal/games"
	"github.com/fadedpez/tucoramirez/internal/types"
	"github.com/fadedpez/tucoramirez/pkg/cards"
	"github.com/fadedpez/tucoramirez/pkg/games/common"
	"github.com/fadedpez/tucoramirez/pkg/storage"
	"github.com/google/uuid"
)

const (
	MaxPlayers = 4
)

type GameState string

const (
	StateWaiting  GameState = "waiting"
	StatePlaying  GameState = "playing"
	StateFinished GameState = "finished"
)

// Game represents a single blackjack game
type Game struct {
	ID          string
	CreatorID   string
	ChannelID   string
	MessageID   string
	Players     map[string]*common.Player
	DealerHand  []cards.Card
	Deck        *cards.Deck
	State       GameState
	CurrentTurn int
	mu          sync.RWMutex
}

// Ensure Game implements the games.Game interface
var _ games.Game = (*Game)(nil)

// NewGame creates a new blackjack game
func NewGame(creatorID, channelID string) *Game {
	return &Game{
		ID:          uuid.New().String(),
		CreatorID:   creatorID,
		ChannelID:   channelID,
		Players:     make(map[string]*common.Player),
		State:       StateWaiting,
		Deck:        cards.NewDeck(),
		DealerHand:  make([]cards.Card, 0),
		CurrentTurn: 0,
		mu:          sync.RWMutex{},
	}
}

// NewGameFromState creates a new game from a stored state
func NewGameFromState(state *storage.GameState) *Game {
	game := NewGame(state.CreatorID, state.ChannelID)
	game.ID = state.ID

	var gameState struct {
		Players     map[string]*common.Player `json:"players"`
		DealerHand  []cards.Card             `json:"dealer_hand"`
		State       GameState                `json:"state"`
		CurrentTurn int                      `json:"current_turn"`
	}

	if err := json.Unmarshal(state.State, &gameState); err != nil {
		return game
	}

	game.Players = gameState.Players
	game.DealerHand = gameState.DealerHand
	game.State = gameState.State
	game.CurrentTurn = gameState.CurrentTurn

	return game
}

// AddPlayer adds a player to the game
func (g *Game) AddPlayer(player *common.Player) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.Players) >= MaxPlayers {
		return types.NewGameError(types.ErrTooManyPlayers, fmt.Sprintf("Game is full (%d/%d players)", len(g.Players), MaxPlayers))
	}

	if _, exists := g.Players[player.ID]; exists {
		return types.NewGameError(types.ErrAlreadyJoined, "You're already in the game")
	}

	g.Players[player.ID] = player
	return nil
}

// HandleJoin handles a player joining the game
func (g *Game) HandleJoin(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	// Only allow joining if the game is in waiting state
	if g.State != StateWaiting {
		discord.SendErrorResponse(s, i, fmt.Errorf("cannot join game in progress"))
		return
	}

	// Add the player to the game
	player := common.NewPlayer(i.Member.User.ID)
	player.Username = i.Member.User.Username
	if err := g.AddPlayer(player); err != nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("failed to add player to game: %w", err))
		return
	}

	// Update the game display
	if err := g.UpdateDisplay(s, i); err != nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("failed to update game display: %w", err))
		return
	}
}

// HandleStart handles starting the game
func (g *Game) HandleStart(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	// Only allow the creator to start the game
	if i.Member.User.ID != g.CreatorID {
		discord.SendErrorResponse(s, i, fmt.Errorf("only the creator can start the game"))
		return
	}

	// Start the game
	if err := g.Start(); err != nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("failed to start game: %w", err))
		return
	}

	// Update the game display
	if err := g.UpdateDisplay(s, i); err != nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("failed to update game display: %w", err))
		return
	}
}

// HandleButton handles button interactions
func (g *Game) HandleButton(s discord.SessionHandler, i *discordgo.InteractionCreate) {
	// Get the button ID
	customID := i.MessageComponentData().CustomID
	switch customID {
	case "blackjack_join":
		g.HandleJoin(s, i)
	case "blackjack_start":
		g.HandleStart(s, i)
	case "blackjack_hit":
		if err := g.HandleHit(s, i); err != nil {
			discord.SendErrorResponse(s, i, err)
			return
		}
	case "blackjack_stand":
		if err := g.HandleStand(s, i); err != nil {
			discord.SendErrorResponse(s, i, err)
			return
		}
	case "blackjack_playagain":
		// Just delegate to HandlePlayAgain, let manager handle game replacement
		_, err := g.HandlePlayAgain(s, i)
		if err != nil {
			discord.SendErrorResponse(s, i, err)
			return
		}
		// Don't update display here, HandlePlayAgain already does it
		return
	default:
		discord.SendErrorResponse(s, i, fmt.Errorf("unknown button: %s", customID))
		return
	}

	// Update game display
	if err := discord.UpdateGameResponse(s, i, g.String(), g.GetButtons()); err != nil {
		discord.SendErrorResponse(s, i, fmt.Errorf("failed to update game: %w", err))
		return
	}
}

// HandleHit handles a player hitting
func (g *Game) HandleHit(s discord.SessionHandler, i *discordgo.InteractionCreate) error {
	// First validate without any locks
	if err := g.ValidatePlayerAction(i.Member.User.ID); err != nil {
		discord.SendErrorResponse(s, i, err)
		return err
	}

	// Then take the write lock for state changes
	g.mu.Lock()
	player := g.Players[i.Member.User.ID]
	card := g.Deck.DrawOne()
	player.AddCard(card)
	player.SetScore(calculateScore(player.GetHand()))

	if player.GetScore() > 21 {
		player.Stand()
		g.nextPlayer()
	}
	g.mu.Unlock()

	// Update game message
	return discord.UpdateGameResponse(s, i, g.String(), g.GetButtons())
}

// HandleStand handles a player standing
func (g *Game) HandleStand(s discord.SessionHandler, i *discordgo.InteractionCreate) error {
	// First validate without any locks
	if err := g.ValidatePlayerAction(i.Member.User.ID); err != nil {
		discord.SendErrorResponse(s, i, err)
		return err
	}

	// Then take the write lock for state changes
	g.mu.Lock()
	player := g.Players[i.Member.User.ID]
	player.Stand()
	g.nextPlayer()
	g.mu.Unlock()

	// Update game message
	return discord.UpdateGameResponse(s, i, g.String(), g.GetButtons())
}

// Start begins the game
func (g *Game) Start() error {
	if len(g.Players) < 1 {
		return types.NewGameError(types.ErrNotEnoughPlayers, "Need at least 1 player to start")
	}

	if g.State != StateWaiting {
		return types.NewGameError(types.ErrGameInProgress, "Game has already started")
	}

	g.State = StatePlaying
	g.Deck.Shuffle()

	// Deal initial cards
	for _, player := range g.Players {
		card1 := g.Deck.DrawOne()
		card2 := g.Deck.DrawOne()
		player.AddCard(card1)
		player.AddCard(card2)
		player.SetScore(calculateScore(player.GetHand()))
	}

	// Deal dealer's cards
	card1 := g.Deck.DrawOne()
	card2 := g.Deck.DrawOne()
	g.DealerHand = []cards.Card{card1, card2}

	return nil
}

// IsFinished checks if the game is over
func (g *Game) IsFinished() bool {
	return g.State == StateFinished
}

// ValidatePlayerAction checks if the player can perform an action
func (g *Game) ValidatePlayerAction(playerID string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	player, exists := g.Players[playerID]
	if !exists {
		return types.NewGameError(types.ErrPlayerNotFound, "You are not in this game")
	}

	if player.Busted {
		return types.NewGameError(types.ErrPlayerBusted, "You have already busted")
	}

	if player.Stood {
		return types.NewGameError(types.ErrPlayerStanding, "You have already stood")
	}

	currentPlayer := g.getCurrentPlayer()
	if currentPlayer == nil || currentPlayer.ID != playerID {
		return types.NewGameError(types.ErrNotPlayerTurn, "It's not your turn")
	}

	return nil
}

// GetButtons returns the appropriate button components for the current game state
func (g *Game) GetButtons() []discordgo.MessageComponent {
	if g.State == StateWaiting {
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Join Game",
						Style:    discordgo.SuccessButton,
						CustomID: "blackjack_join",
						Disabled: len(g.Players) >= MaxPlayers,
					},
					discordgo.Button{
						Label:    "Start Game",
						Style:    discordgo.PrimaryButton,
						CustomID: "blackjack_start",
					},
				},
			},
		}
	}

	if g.State == StatePlaying {
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
						Style:    discordgo.DangerButton,
						CustomID: "blackjack_stand",
					},
				},
			},
		}
	}

	if g.State == StateFinished {
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Play Again",
						Style:    discordgo.SuccessButton,
						CustomID: "blackjack_playagain",
					},
				},
			},
		}
	}

	return nil
}

// String returns a string representation of the game state
func (g *Game) String() string {
	var sb strings.Builder

	// Show dealer's hand
	sb.WriteString("🎰 Dealer's Hand:\n")
	if g.State == StatePlaying {
		// Show only first card during play
		sb.WriteString(fmt.Sprintf("  %s, [?]\n", g.DealerHand[0]))
	} else {
		// Show all cards at end
		dealerScore := calculateScore(g.DealerHand)
		sb.WriteString(fmt.Sprintf("  %s (Score: %d)\n", formatHand(g.DealerHand), dealerScore))
	}

	// Show players' hands
	sb.WriteString("\n👥 Players:\n")
	if g.State == StateWaiting {
		sb.WriteString(fmt.Sprintf("Waiting for players (%d/%d):\n", len(g.Players), MaxPlayers))
		for _, p := range g.Players {
			sb.WriteString(fmt.Sprintf("  %s\n", p.Username))
		}
	} else {
		currentPlayerID := g.getCurrentPlayerID()
		for _, p := range g.Players {
			marker := " "
			if p.ID == currentPlayerID {
				marker = ">"
			}
			sb.WriteString(fmt.Sprintf("%s %s: %s (Score: %d)\n", marker, p.Username, formatHand(p.GetHand()), p.GetScore()))
		}
	}

	return sb.String()
}

// MarshalState returns a JSON representation of the game state
func (g *Game) MarshalState() json.RawMessage {
	g.mu.RLock()
	defer g.mu.RUnlock()

	state := struct {
		Players     map[string]*common.Player `json:"players"`
		DealerHand  []cards.Card             `json:"dealer_hand"`
		State       GameState                `json:"state"`
		CurrentTurn int                      `json:"current_turn"`
	}{
		Players:     g.Players,
		DealerHand:  g.DealerHand,
		State:       g.State,
		CurrentTurn: g.CurrentTurn,
	}

	data, err := json.Marshal(state)
	if err != nil {
		return nil
	}
	return json.RawMessage(data)
}

// LoadState loads the game state from a JSON representation
func (g *Game) LoadState(data json.RawMessage) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	state := struct {
		Players     map[string]*common.Player `json:"players"`
		DealerHand  []cards.Card             `json:"dealer_hand"`
		State       GameState                `json:"state"`
		CurrentTurn int                      `json:"current_turn"`
	}{}

	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal game state: %w", err)
	}

	g.Players = state.Players
	g.DealerHand = state.DealerHand
	g.State = state.State
	g.CurrentTurn = state.CurrentTurn

	return nil
}

// HandlePlayAgain handles when a player wants to play another game
func (g *Game) HandlePlayAgain(s discord.SessionHandler, i *discordgo.InteractionCreate) (*Game, error) {
	// Only allow play again if the game is finished
	if g.State != StateFinished {
		return nil, fmt.Errorf("game must be finished before starting a new one")
	}

	// Only allow the player who clicked to start a new game as the creator
	playerID := i.Member.User.ID

	// Create a new game with this player as the creator
	newGame := NewGame(playerID, g.ChannelID)

	// Add the creator as the first player
	player := common.NewPlayer(playerID)
	player.Username = i.Member.User.Username
	if err := newGame.AddPlayer(player); err != nil {
		return nil, fmt.Errorf("failed to add player to new game: %w", err)
	}

	return newGame, nil
}

// UpdateDisplay updates the game display in Discord
func (g *Game) UpdateDisplay(s discord.SessionHandler, i *discordgo.InteractionCreate) error {
	return discord.UpdateGameResponse(s, i, g.String(), g.GetButtons())
}

// Helper functions
func formatHand(hand []cards.Card) string {
	cards := make([]string, len(hand))
	for i, card := range hand {
		cards[i] = card.String()
	}
	return strings.Join(cards, ", ")
}

func calculateScore(hand []cards.Card) int {
	score := 0
	aces := 0

	for _, card := range hand {
		if card.Rank == cards.Ace {
			aces++
		} else {
			switch card.Rank {
			case cards.Two:
				score += 2
			case cards.Three:
				score += 3
			case cards.Four:
				score += 4
			case cards.Five:
				score += 5
			case cards.Six:
				score += 6
			case cards.Seven:
				score += 7
			case cards.Eight:
				score += 8
			case cards.Nine:
				score += 9
			case cards.Ten:
				score += 10
			case cards.Jack:
				score += 10
			case cards.Queen:
				score += 10
			case cards.King:
				score += 10
			}
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

func (g *Game) getCurrentPlayerID() string {
	if len(g.Players) == 0 {
		return ""
	}

	playerIDs := make([]string, 0, len(g.Players))
	for id := range g.Players {
		playerIDs = append(playerIDs, id)
	}

	return playerIDs[g.CurrentTurn%len(playerIDs)]
}

func (g *Game) getCurrentPlayer() *common.Player {
	if g.CurrentTurn >= len(g.Players) {
		return nil
	}

	i := 0
	for _, player := range g.Players {
		if i == g.CurrentTurn {
			return player
		}
		i++
	}

	return nil
}

func (g *Game) nextPlayer() {
	g.CurrentTurn++

	// Check if all players have stood
	allStood := true
	for _, p := range g.Players {
		if !p.HasStood() {
			allStood = false
			break
		}
	}

	if allStood {
		g.finishGame()
	}
}

func (g *Game) finishGame() {
	g.State = StateFinished

	// Play dealer's hand
	dealerScore := calculateScore(g.DealerHand)
	for dealerScore < 17 {
		card := g.Deck.DrawOne()
		g.DealerHand = append(g.DealerHand, card)
		dealerScore = calculateScore(g.DealerHand)
	}
}
