package entities

import "time"

// PlayerStatistics represents aggregated statistics for a player in a specific game type
type PlayerStatistics struct {
	PlayerID      string
	GameType      GameState
	GamesPlayed   int
	Wins          int
	Losses        int
	Pushes        int
	Blackjacks    int
	Busts         int
	Splits        int
	DoubleDowns   int
	Insurances    int
	TotalBet      int64
	TotalWinnings int64
	LastUpdated   time.Time
}

// NetProfit calculates the player's net profit
func (s *PlayerStatistics) NetProfit() int64 {
	return s.TotalWinnings - s.TotalBet
}

// WinRate calculates the player's win rate as a percentage
func (s *PlayerStatistics) WinRate() float64 {
	if s.GamesPlayed == 0 {
		return 0.0
	}
	return float64(s.Wins) / float64(s.GamesPlayed) * 100.0
}
