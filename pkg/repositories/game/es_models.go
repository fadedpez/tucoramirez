package game

import (
	"time"
)

// ESGameResult represents a game result document in Elasticsearch
type ESGameResult struct {
	GameID      string    `json:"game_id"`
	GameType    string    `json:"game_type"`
	ChannelID   string    `json:"channel_id"`
	CompletedAt time.Time `json:"completed_at"`
	Players     []ESPlayerResult `json:"players"`
	DealerCards []string  `json:"dealer_cards"`
	DealerScore int       `json:"dealer_score"`
}

// ESPlayerResult represents a player result in Elasticsearch
type ESPlayerResult struct {
	PlayerID       string `json:"player_id"`
	HandID         string `json:"hand_id"`
	ParentHandID   string `json:"parent_hand_id,omitempty"`
	Bet            int64  `json:"bet"`
	Winnings       int64  `json:"winnings"`
	Result         string `json:"result"` // "win", "loss", "push"
	Score          int    `json:"score"`
	Cards          []string `json:"cards"`
	Blackjack      bool   `json:"blackjack"`
	Busted         bool   `json:"busted"`
	HasSplit       bool   `json:"has_split"`
	IsDoubledDown  bool   `json:"is_doubled_down"`
	DoubleDownBet  int64  `json:"double_down_bet,omitempty"`
	HasInsurance   bool   `json:"has_insurance"`
	InsuranceBet   int64  `json:"insurance_bet,omitempty"`
	InsurancePayout int64  `json:"insurance_payout,omitempty"`
	Actions        []string `json:"actions"`
}
