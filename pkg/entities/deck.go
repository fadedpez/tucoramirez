package entities

import (
	"math/rand"
	"time"
)

type Deck struct {
	Cards []*Card
}

// NewDeck creates a new deck of 52 cards, one of each rank and suit
func NewDeck() *Deck {
	cards := make([]*Card, 0, 52)
	suits := []Suit{Hearts, Diamonds, Clubs, Spades}
	ranks := []Rank{Ace, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King}

	for _, suit := range suits {
		for _, rank := range ranks {
			cards = append(cards, NewCard(suit, rank))
		}
	}

	return &Deck{Cards: cards}
}

func (d *Deck) Shuffle() {
	// Create a new random source using current time as seed
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Use Go's built-in shuffle algorithm
	r.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

// Draw removes and returns the top card from the deck
func (d *Deck) Draw() *Card {
	if len(d.Cards) == 0 {
		return nil
	}
	card := d.Cards[0]
	d.Cards = d.Cards[1:]
	return card
}
