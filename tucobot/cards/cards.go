package cards

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// Card represents a playing card with a rank and suit
type Card struct {
	Rank string `json:"rank"`
	Suit string `json:"suit"`
}

func (c Card) String() string {
	return fmt.Sprintf("%s%s", c.Rank, c.Suit)
}

func (c Card) Value() int {
	switch c.Rank {
	case "A":
		return 11
	case "K", "Q", "J":
		return 10
	default:
		if c.Rank == "10" {
			return 10
		}
		// For numeric cards (2-9), convert the first character to int
		return int(c.Rank[0] - '0')
	}
}

// Deck represents a collection of cards
type Deck struct {
	Cards []Card `json:"cards"`
}

// NewDeck creates a new deck of 52 cards
func NewDeck() (*Deck, error) {
	ranks := []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	suits := []string{"♠", "♥", "♦", "♣"}
	cards := make([]Card, 0, len(ranks)*len(suits))

	for _, suit := range suits {
		for _, rank := range ranks {
			cards = append(cards, Card{Rank: rank, Suit: suit})
		}
	}

	if len(cards) == 0 {
		return nil, fmt.Errorf("failed to create deck")
	}
	return &Deck{Cards: cards}, nil
}

// Shuffle randomizes the order of cards in the deck
func (d *Deck) Shuffle() error {
	if d == nil || len(d.Cards) == 0 {
		return fmt.Errorf("deck is empty or nil")
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
	return nil
}

// Draw removes and returns the top card from the deck
func (d *Deck) Draw() (Card, error) {
	if d == nil || len(d.Cards) == 0 {
		return Card{}, fmt.Errorf("deck is empty or nil")
	}
	card := d.Cards[0]
	d.Cards = d.Cards[1:]
	return card, nil
}

func (d *Deck) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Cards []Card `json:"cards"`
	}{
		Cards: d.Cards,
	})
}
