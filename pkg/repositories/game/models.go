package game

import (
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

// GameRecord represents a record of a completed blackjack game
type GameRecord struct {
	ID            string       `json:"id" bson:"_id"`
	GameType      string       `json:"game_type" bson:"game_type"`
	ChannelID     string       `json:"channel_id" bson:"channel_id"`
	StartTime     time.Time    `json:"start_time" bson:"start_time"`
	EndTime       time.Time    `json:"end_time" bson:"end_time"`
	PlayerRecords []HandRecord `json:"player_records" bson:"player_records"`
	DealerCards   []string     `json:"dealer_cards" bson:"dealer_cards"`
	DealerScore   int          `json:"dealer_score" bson:"dealer_score"`
}

// HandRecord represents a record of a player's hand in a blackjack game
type HandRecord struct {
	PlayerID      string                 `json:"player_id" bson:"player_id"`
	HandID        string                 `json:"hand_id" bson:"hand_id"`
	ParentHandID  string                 `json:"parent_hand_id,omitempty" bson:"parent_hand_id,omitempty"`
	Cards         []string               `json:"cards" bson:"cards"`
	FinalScore    int                    `json:"final_score" bson:"final_score"`
	InitialBet    int64                  `json:"initial_bet" bson:"initial_bet"`
	IsSplit       bool                   `json:"is_split" bson:"is_split"`
	IsDoubledDown bool                   `json:"is_doubled_down" bson:"is_doubled_down"`
	DoubleDownBet int64                  `json:"double_down_bet,omitempty" bson:"double_down_bet,omitempty"`
	HasInsurance  bool                   `json:"has_insurance" bson:"has_insurance"`
	InsuranceBet  int64                  `json:"insurance_bet,omitempty" bson:"insurance_bet,omitempty"`
	Result        entities.Result        `json:"result" bson:"result"`
	Payout        int64                  `json:"payout" bson:"payout"`
	InsurancePayout int64                `json:"insurance_payout,omitempty" bson:"insurance_payout,omitempty"`
	Actions       []string               `json:"actions" bson:"actions"`
	Metadata      map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
}
