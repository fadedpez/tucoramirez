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

// Metadata keys for special betting options
const (
	MetaKeyDoubledDown  = "doubled_down"   // Whether the hand has been doubled down
	MetaKeyDoubleDownBet = "double_down_bet" // The amount of the double down bet
	MetaKeySplit        = "is_split"        // Whether the hand is a split hand
	MetaKeySplitHandID  = "split_hand_id"   // ID of the split hand (for tracking split pairs)
	MetaKeyParentHandID = "parent_hand_id"  // ID of the parent hand (for split hands)
	MetaKeyInsurance    = "has_insurance"   // Whether insurance was taken
	MetaKeyInsuranceBet = "insurance_bet"   // The amount of the insurance bet
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
	Cards    []*entities.Card
	Status   Status
	Metadata map[string]interface{} // For special bet data like doubled down, split, etc.
}

// NewHand creates a new blackjack hand
func NewHand() *Hand {
	return &Hand{
		Cards:    make([]*entities.Card, 0),
		Status:   StatusPlaying,
		Metadata: make(map[string]interface{}),
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

// Metadata helper functions

// IsDoubledDown checks if a hand has been doubled down
func (h *Hand) IsDoubledDown() bool {
	if val, ok := h.Metadata[MetaKeyDoubledDown]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}

// SetDoubledDown sets the doubled down status of a hand
func (h *Hand) SetDoubledDown(value bool) {
	if h.Metadata == nil {
		h.Metadata = make(map[string]interface{})
	}
	h.Metadata[MetaKeyDoubledDown] = value
}

// GetDoubleDownBet gets the double down bet amount
func (h *Hand) GetDoubleDownBet() int64 {
	if val, ok := h.Metadata[MetaKeyDoubleDownBet]; ok {
		if intVal, ok := val.(int64); ok {
			return intVal
		}
	}
	return 0
}

// SetDoubleDownBet sets the double down bet amount
func (h *Hand) SetDoubleDownBet(amount int64) {
	if h.Metadata == nil {
		h.Metadata = make(map[string]interface{})
	}
	h.Metadata[MetaKeyDoubleDownBet] = amount
}

// IsSplit checks if a hand is a split hand
func (h *Hand) IsSplit() bool {
	if val, ok := h.Metadata[MetaKeySplit]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}

// SetSplit sets the split status of a hand
func (h *Hand) SetSplit(value bool) {
	if h.Metadata == nil {
		h.Metadata = make(map[string]interface{})
	}
	h.Metadata[MetaKeySplit] = value
}

// GetSplitHandID gets the ID of the split hand
func (h *Hand) GetSplitHandID() string {
	if val, ok := h.Metadata[MetaKeySplitHandID]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

// SetSplitHandID sets the ID of the split hand
func (h *Hand) SetSplitHandID(id string) {
	if h.Metadata == nil {
		h.Metadata = make(map[string]interface{})
	}
	h.Metadata[MetaKeySplitHandID] = id
}

// GetParentHandID gets the ID of the parent hand
func (h *Hand) GetParentHandID() string {
	if val, ok := h.Metadata[MetaKeyParentHandID]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return ""
}

// SetParentHandID sets the ID of the parent hand
func (h *Hand) SetParentHandID(id string) {
	if h.Metadata == nil {
		h.Metadata = make(map[string]interface{})
	}
	h.Metadata[MetaKeyParentHandID] = id
}

// HasInsurance checks if a hand has insurance
func (h *Hand) HasInsurance() bool {
	if val, ok := h.Metadata[MetaKeyInsurance]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return false
}

// SetInsurance sets the insurance status of a hand
func (h *Hand) SetInsurance(value bool) {
	if h.Metadata == nil {
		h.Metadata = make(map[string]interface{})
	}
	h.Metadata[MetaKeyInsurance] = value
}

// GetInsuranceBet gets the insurance bet amount
func (h *Hand) GetInsuranceBet() int64 {
	if val, ok := h.Metadata[MetaKeyInsuranceBet]; ok {
		if intVal, ok := val.(int64); ok {
			return intVal
		}
	}
	return 0
}

// SetInsuranceBet sets the insurance bet amount
func (h *Hand) SetInsuranceBet(amount int64) {
	if h.Metadata == nil {
		h.Metadata = make(map[string]interface{})
	}
	h.Metadata[MetaKeyInsuranceBet] = amount
}
