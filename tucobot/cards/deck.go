package cards

import (
	"fmt"
	"math/rand"
	"strings"
)

type Suit string
type Rank string

const (
	Hearts   Suit = "♥"
	Diamonds Suit = "♦"
	Clubs    Suit = "♣"
	Spades   Suit = "♠"
)

const (
	Ace   Rank = "A"
	Two   Rank = "2"
	Three Rank = "3"
	Four  Rank = "4"
	Five  Rank = "5"
	Six   Rank = "6"
	Seven Rank = "7"
	Eight Rank = "8"
	Nine  Rank = "9"
	Ten   Rank = "10"
	Jack  Rank = "J"
	Queen Rank = "Q"
	King  Rank = "K"
)

type Card struct {
	Rank Rank
	Suit Suit
}

type Player struct {
	ID    string
	Name  string
	Hand  []Card
	Score int
}

type GameSession struct {
	Players   []Player
	Dealer    Player
	Deck      []Card
	CurrentID string
}

func (c Card) String() string {
	return string(c.Rank) + string(c.Suit)
}

func (c Card) Value() int {
	switch c.Rank {
	case Ace:
		return 11
	case Jack, Queen, King:
		return 10
	default:
		return int(c.Rank[0] - '0')
	}
}

func NewDeck() []Card {
	var deck []Card
	suits := []Suit{Hearts, Diamonds, Clubs, Spades}
	ranks := []Rank{Ace, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King}

	for _, suit := range suits {
		for _, rank := range ranks {
			deck = append(deck, Card{Rank: rank, Suit: suit})
		}
	}

	rand.Shuffle(len(deck), func(i, j int) {
		deck[i], deck[j] = deck[j], deck[i]
	})

	return deck
}

func NewGame() *GameSession {
	return &GameSession{
		Players: make([]Player, 0),
		Dealer: Player{
			ID:   "dealer",
			Name: "Dealer",
			Hand: make([]Card, 0),
		},
		Deck: NewDeck(),
	}
}

func (g *GameSession) AddPlayer(id string) {
	g.Players = append(g.Players, Player{
		ID:   id,
		Hand: make([]Card, 0),
	})
	g.CurrentID = id
}

func (g *GameSession) DealInitialCards() {
	// Deal two cards to each player
	for i := 0; i < 2; i++ {
		for j := range g.Players {
			g.DealCard(g.Players[j].ID)
		}
		// Deal one face-up card to dealer on first round
		if i == 0 {
			g.dealCardToDealer()
		}
	}
}

func (g *GameSession) DealCard(playerID string) {
	if len(g.Deck) == 0 {
		g.Deck = NewDeck()
	}

	var card Card
	card, g.Deck = g.Deck[0], g.Deck[1:]

	for i := range g.Players {
		if g.Players[i].ID == playerID {
			g.Players[i].Hand = append(g.Players[i].Hand, card)
			g.UpdatePlayerScore(&g.Players[i])
			break
		}
	}
}

func (g *GameSession) dealCardToDealer() {
	if len(g.Deck) == 0 {
		g.Deck = NewDeck()
	}

	var card Card
	card, g.Deck = g.Deck[0], g.Deck[1:]
	g.Dealer.Hand = append(g.Dealer.Hand, card)
	g.UpdatePlayerScore(&g.Dealer)
}

func (g *GameSession) UpdatePlayerScore(p *Player) {
	score := 0
	aces := 0

	for _, card := range p.Hand {
		if card.Rank == Ace {
			aces++
		} else {
			score += card.Value()
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

	p.Score = score
}

func (g *GameSession) IsPlayerTurn(playerID string) bool {
	return g.CurrentID == playerID
}

func (g *GameSession) IsPlayerBust(playerID string) bool {
	for _, p := range g.Players {
		if p.ID == playerID {
			return p.Score > 21
		}
	}
	return false
}

func (g *GameSession) DealerPlay() {
	for g.Dealer.Score < 17 {
		g.dealCardToDealer()
	}
}

func (g *GameSession) DetermineWinner() string {
	player := g.Players[0] // We only support one player for now
	
	if player.Score > 21 {
		return "dealer"
	}
	if g.Dealer.Score > 21 {
		return "player"
	}
	if player.Score > g.Dealer.Score {
		return "player"
	}
	if g.Dealer.Score > player.Score {
		return "dealer"
	}
	return "tie"
}

func (g *GameSession) GetGameState() string {
	var sb strings.Builder

	// Show dealer's hand
	sb.WriteString("Dealer's hand: ")
	for i, card := range g.Dealer.Hand {
		if i == 0 || len(g.Dealer.Hand) == len(g.Players[0].Hand) {
			sb.WriteString(card.String() + " ")
		} else {
			sb.WriteString("🂠 ")
		}
	}
	if len(g.Dealer.Hand) == len(g.Players[0].Hand) {
		sb.WriteString(fmt.Sprintf("(Score: %d)", g.Dealer.Score))
	}
	sb.WriteString("\n")

	// Show player's hand
	for _, p := range g.Players {
		sb.WriteString(fmt.Sprintf("<@%s>'s hand: ", p.ID))
		for _, card := range p.Hand {
			sb.WriteString(card.String() + " ")
		}
		sb.WriteString(fmt.Sprintf("(Score: %d)", p.Score))
		sb.WriteString("\n")
	}

	return sb.String()
}
