package cards

import (
	"math/rand"
	"time"
)

// Suit represents a card suit
type Suit string

const (
	Hearts   Suit = "♥"
	Diamonds Suit = "♦"
	Clubs    Suit = "♣"
	Spades   Suit = "♠"
)

// Rank represents a card rank
type Rank string

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

// Card represents a playing card
type Card struct {
	Suit Suit
	Rank Rank
}

// String returns a string representation of the card
func (c Card) String() string {
	return string(c.Suit) + string(c.Rank)
}

// Deck represents a deck of cards
type Deck struct {
	Cards []Card
}

// NewDeck creates a new deck of cards
func NewDeck() *Deck {
	deck := &Deck{}
	suits := []Suit{Hearts, Diamonds, Clubs, Spades}
	ranks := []Rank{Ace, Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King}

	for _, suit := range suits {
		for _, rank := range ranks {
			deck.Cards = append(deck.Cards, Card{Suit: suit, Rank: rank})
		}
	}

	return deck
}

// Shuffle shuffles the deck
func (d *Deck) Shuffle() {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(d.Cards), func(i, j int) {
		d.Cards[i], d.Cards[j] = d.Cards[j], d.Cards[i]
	})
}

// Draw draws n cards from the deck
func (d *Deck) Draw(n int) []Card {
	if n > len(d.Cards) {
		n = len(d.Cards)
	}

	cards := d.Cards[:n]
	d.Cards = d.Cards[n:]
	return cards
}

// DrawOne draws one card from the deck
func (d *Deck) DrawOne() Card {
	cards := d.Draw(1)
	if len(cards) == 0 {
		return Card{}
	}
	return cards[0]
}
