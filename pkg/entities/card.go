package entities

import "fmt"

// Suit represents a card suit

type Suit string

const (
	Hearts   Suit = "HEARTS"
	Diamonds Suit = "DIAMONDS"
	Clubs    Suit = "CLUBS"
	Spades   Suit = "SPADES"
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

// NewCard creates a new card

func NewCard(suit Suit, rank Rank) *Card {
	return &Card{
		Suit: suit,
		Rank: rank,
	}
}

// String returns the string representation of the card

func (c *Card) String() string {
	return fmt.Sprintf("%s of %s", c.Rank, c.Suit)
}
