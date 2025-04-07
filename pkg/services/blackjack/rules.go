package blackjack

import (
	"strconv"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

const (
	StandardDecks      = 6  // Standard number of decks in the shoe
	ReshuffleThreshold = 75 // Reshuffle when 75 cards remain (~25% of one shoe)
	MaxPlayers         = 7  // Max number of players allowed in a blackjack game
)

// Result represents the outcome of a blackjack hand
type Result string

const (
	ResultWin       Result = "WIN"
	ResultLose      Result = "LOSE"
	ResultPush      Result = "PUSH"
	ResultBlackjack Result = "BLACKJACK"
)

// String returns the string representation of the result
func (r Result) String() string {
	return string(r)
}

// IsWin returns true if this result represents a win
func (r Result) IsWin() bool {
	return r == ResultWin || r == ResultBlackjack
}

func GetCardValue(card *entities.Card) int {
	switch card.Rank {
	case entities.Ace:
		return 11
	case entities.Jack, entities.Queen, entities.King:
		return 10
	default:
		val, _ := strconv.Atoi(string(card.Rank))
		return val
	}
}

func IsAce(card *entities.Card) bool {
	return card.Rank == entities.Ace
}

func GetBestScore(cards []*entities.Card) int {
	score := 0
	aces := 0

	// First count non-aces

	for _, card := range cards {
		if IsAce(card) {
			aces++
		} else {
			score += GetCardValue(card)
		}
	}

	// Then handle aces

	for i := 0; i < aces; i++ {
		if score+11 <= 21 {
			score += 11
		} else {
			score += 1
		}
	}

	return score
}

func IsBlackjack(cards []*entities.Card) bool {
	return len(cards) == 2 && GetBestScore(cards) == 21
}

// IsBust checks if a hand exceeds 21
func IsBust(cards []*entities.Card) bool {
	return GetBestScore(cards) > 21
}

// CompareHands compares two hands and returns:
// 1 if hand1 wins
// -1 if hand 2 wins
// 0 if push (tie)
func CompareHands(hand1, hand2 []*entities.Card) int {
	// Handle blackjacks
	bj1 := IsBlackjack(hand1)
	bj2 := IsBlackjack(hand2)
	if bj1 && !bj2 {
		return 1
	} else if !bj1 && bj2 {
		return -1
	} else if bj1 && bj2 {
		return 0
	}

	// Handle busts
	bust1 := IsBust(hand1)
	bust2 := IsBust(hand2)
	if bust1 && !bust2 {
		return -1
	} else if !bust1 && bust2 {
		return 1
	} else if bust1 && bust2 {
		return 0
	}

	// Compare scores
	score1 := GetBestScore(hand1)
	score2 := GetBestScore(hand2)
	if score1 > score2 {
		return 1
	} else if score1 < score2 {
		return -1
	}
	return 0
}

// NewBlackjackDeck creates a new shuffled shoe with a standard number of decks
func NewBlackjackDeck() *entities.Deck {
	deck := entities.NewDeck()

	// Add more decks to meet the StandardDecks size definition
	for i := 1; i < StandardDecks; i++ {
		deck.Cards = append(deck.Cards, entities.NewDeck().Cards...)
	}

	deck.Shuffle()
	return deck
}

// ShouldReshuffle checks if the deck should be reshuffled based on the remaining cards
func ShouldReshuffle(deck *entities.Deck) bool {
	return len(deck.Cards) < ReshuffleThreshold
}
