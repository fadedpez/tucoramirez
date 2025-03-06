package common

import "github.com/fadedpez/tucoramirez/pkg/cards"

// Player represents a player in any game
type Player struct {
	ID       string
	Username string
	Hand     []cards.Card
	Score    int
	Stood    bool
	Busted   bool
}

// NewPlayer creates a new player with the given ID
func NewPlayer(id string) *Player {
	return &Player{
		ID:   id,
		Hand: make([]cards.Card, 0),
	}
}

// ClearHand removes all cards from the player's hand
func (p *Player) ClearHand() {
	p.Hand = []cards.Card{}
	p.Score = 0
	p.Stood = false
	p.Busted = false
}

// AddCard adds a card to the player's hand
func (p *Player) AddCard(card cards.Card) {
	p.Hand = append(p.Hand, card)
}

// GetHand returns the player's current hand
func (p *Player) GetHand() []cards.Card {
	return p.Hand
}

// SetScore sets the player's current score
func (p *Player) SetScore(score int) {
	p.Score = score
}

// GetScore returns the player's current score
func (p *Player) GetScore() int {
	return p.Score
}

// Stand marks the player as standing
func (p *Player) Stand() {
	p.Stood = true
}

// HasStood returns whether the player has stood
func (p *Player) HasStood() bool {
	return p.Stood
}

// Bust marks the player as busted
func (p *Player) Bust() {
	p.Busted = true
}

// HasBusted returns whether the player has busted
func (p *Player) HasBusted() bool {
	return p.Busted
}
