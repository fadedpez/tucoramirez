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
	PlayerID        string                 `json:"player_id" bson:"player_id"`
	HandID          string                 `json:"hand_id" bson:"hand_id"`
	ParentHandID    string                 `json:"parent_hand_id,omitempty" bson:"parent_hand_id,omitempty"`
	Cards           []string               `json:"cards" bson:"cards"`
	FinalScore      int                    `json:"final_score" bson:"final_score"`
	InitialBet      int64                  `json:"initial_bet" bson:"initial_bet"`
	IsSplit         bool                   `json:"is_split" bson:"is_split"`
	IsDoubledDown   bool                   `json:"is_doubled_down" bson:"is_doubled_down"`
	DoubleDownBet   int64                  `json:"double_down_bet,omitempty" bson:"double_down_bet,omitempty"`
	HasInsurance    bool                   `json:"has_insurance" bson:"has_insurance"`
	InsuranceBet    int64                  `json:"insurance_bet,omitempty" bson:"insurance_bet,omitempty"`
	Result          entities.Result        `json:"result" bson:"result"`
	Payout          int64                  `json:"payout" bson:"payout"`
	InsurancePayout int64                  `json:"insurance_payout,omitempty" bson:"insurance_payout,omitempty"`
	Actions         []string               `json:"actions" bson:"actions"`
	Metadata        map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

// ToESGameResult converts a GameRecord to an ESGameResult
func (g *GameRecord) ToESGameResult() *ESGameResult {
	esResult := &ESGameResult{
		GameID:      g.ID,
		GameType:    g.GameType,
		ChannelID:   g.ChannelID,
		CompletedAt: g.EndTime,
		DealerCards: g.DealerCards,
		DealerScore: g.DealerScore,
		Players:     make([]ESPlayerResult, 0, len(g.PlayerRecords)),
	}

	// Convert each hand record to an ES player result
	for _, handRecord := range g.PlayerRecords {
		esResult.Players = append(esResult.Players, handRecord.ToESPlayerResult())
	}

	return esResult
}

// ToESPlayerResult converts a HandRecord to an ESPlayerResult
func (h *HandRecord) ToESPlayerResult() ESPlayerResult {
	// Calculate total winnings (payout + insurance payout)
	winnings := h.Payout
	if h.HasInsurance {
		winnings += h.InsurancePayout
	}

	// Calculate total bet (initial bet + double down bet + insurance bet)
	bet := h.InitialBet
	if h.IsDoubledDown {
		bet += h.DoubleDownBet
	}
	if h.HasInsurance {
		bet += h.InsuranceBet
	}

	// Determine if the hand had blackjack or busted
	blackjack := false
	busted := false

	// In blackjack, a natural 21 with 2 cards is a blackjack
	if h.FinalScore == 21 && len(h.Cards) == 2 {
		blackjack = true
	}

	// In blackjack, a score over 21 is a bust
	if h.FinalScore > 21 {
		busted = true
	}

	return ESPlayerResult{
		PlayerID:        h.PlayerID,
		HandID:          h.HandID,
		ParentHandID:    h.ParentHandID,
		Bet:             bet,
		Winnings:        winnings,
		Result:          h.Result.String(),
		Score:           h.FinalScore,
		Cards:           h.Cards,
		Blackjack:       blackjack,
		Busted:          busted,
		HasSplit:        h.IsSplit,
		IsDoubledDown:   h.IsDoubledDown,
		DoubleDownBet:   h.DoubleDownBet,
		HasInsurance:    h.HasInsurance,
		InsuranceBet:    h.InsuranceBet,
		InsurancePayout: h.InsurancePayout,
		Actions:         h.Actions,
	}
}

// GameResultToESGameResult converts an entities.GameResult to an ESGameResult
func GameResultToESGameResult(result *entities.GameResult) *ESGameResult {
	// Generate a unique ID for the game result
	gameID := "game_" + result.ChannelID + "_" + result.CompletedAt.Format(time.RFC3339)
	
	esResult := &ESGameResult{
		GameID:      gameID,
		GameType:    string(result.GameType),
		ChannelID:   result.ChannelID,
		CompletedAt: result.CompletedAt,
		Players:     make([]ESPlayerResult, 0, len(result.PlayerResults)),
	}

	// Extract dealer information from game details if available
	if details, ok := result.Details.(interface{ GetDealerCards() []string }); ok {
		esResult.DealerCards = details.GetDealerCards()
	}
	
	if details, ok := result.Details.(interface{ GetDealerScore() int }); ok {
		esResult.DealerScore = details.GetDealerScore()
	}

	// Convert player results
	for _, playerResult := range result.PlayerResults {
		// Create a basic player result with available fields
		esPlayerResult := ESPlayerResult{
			PlayerID:  playerResult.PlayerID,
			Bet:       playerResult.Bet,
			Winnings:  playerResult.Payout,
			Result:    playerResult.Result.String(),
			Score:     playerResult.Score,
		}

		// Extract additional information from metadata if available
		if playerResult.Metadata != nil {
			// Extract hand ID
			if handID, ok := playerResult.Metadata["hand_id"].(string); ok {
				esPlayerResult.HandID = handID
			}
			
			// Extract parent hand ID for split hands
			if parentID, ok := playerResult.Metadata["parent_hand_id"].(string); ok {
				esPlayerResult.ParentHandID = parentID
			}
			
			// Extract cards
			if cards, ok := playerResult.Metadata["cards"].([]string); ok {
				esPlayerResult.Cards = cards
			}
			
			// Extract action history
			if actions, ok := playerResult.Metadata["actions"].([]string); ok {
				esPlayerResult.Actions = actions
			}
			
			// Extract special bet flags
			if blackjack, ok := playerResult.Metadata["blackjack"].(bool); ok {
				esPlayerResult.Blackjack = blackjack
			}
			
			if busted, ok := playerResult.Metadata["busted"].(bool); ok {
				esPlayerResult.Busted = busted
			}
			
			if hasSplit, ok := playerResult.Metadata["split"].(bool); ok {
				esPlayerResult.HasSplit = hasSplit
			}
			
			if doubledDown, ok := playerResult.Metadata["doubled_down"].(bool); ok {
				esPlayerResult.IsDoubledDown = doubledDown
			}
			
			if doubleDownBet, ok := playerResult.Metadata["double_down_bet"].(int64); ok {
				esPlayerResult.DoubleDownBet = doubleDownBet
			}
			
			if hasInsurance, ok := playerResult.Metadata["insurance"].(bool); ok {
				esPlayerResult.HasInsurance = hasInsurance
			}
			
			if insuranceBet, ok := playerResult.Metadata["insurance_bet"].(int64); ok {
				esPlayerResult.InsuranceBet = insuranceBet
			}
			
			if insurancePayout, ok := playerResult.Metadata["insurance_payout"].(int64); ok {
				esPlayerResult.InsurancePayout = insurancePayout
			}
		}

		esResult.Players = append(esResult.Players, esPlayerResult)
	}

	return esResult
}
