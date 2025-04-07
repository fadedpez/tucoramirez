package blackjack

import (
	"errors"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

var (
	ErrHandBust    = errors.New("hand is bust")
	ErrHandStand   = errors.New("hand is stand")
	ErrInvalidCard = errors.New("invalid card")
)

// Status represents the current state of the hand
type Status string

const (
	StatusPlaying Status = "PLAYING"
	StatusBust    Status = "BUST"
	StatusStand   Status = "STAND"
)

// Hand represents a player's hand in a game of blackjack

type Hand struct {
	Cards  []*entities.Card
	Status Status
}

// NewHand creates a new blackjack hand
func NewHand() *Hand {
	return &Hand{
		Cards:  make([]*entities.Card, 0),
		Status: StatusPlaying,
	}
}

// AddCard adds a card to the hand
func (h *Hand) AddCard(card *entities.Card) error {
	if h.Status != StatusPlaying {
		switch h.Status {
		case StatusBust:
			return ErrHandBust
		case StatusStand:
			return ErrHandStand
		}
	}

	if card == nil {
		return ErrInvalidCard
	}

	h.Cards = append(h.Cards, card)

	// Auto-bust if score exceeds 21
	if GetBestScore(h.Cards) > 21 {
		h.Status = StatusBust
		return nil
	}

	return nil
}

// Stand marks the hand as stood
func (h *Hand) Stand() error {
	if h.Status != StatusPlaying {
		switch h.Status {
		case StatusBust:
			return ErrHandBust
		case StatusStand:
			return ErrHandStand
		}
	}

	h.Status = StatusStand
	return nil
}

// Value returns the best possible score for the hand
func (h *Hand) Value() int {
	return GetBestScore(h.Cards)
}
