package statistics

import (
	"context"
	"sort"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
	"github.com/fadedpez/tucoramirez/pkg/repositories/game"
)

// Service provides methods for retrieving and processing player statistics
type Service struct {
	repository game.Repository
}

// NewService creates a new statistics service
func NewService(repository game.Repository) *Service {
	return &Service{
		repository: repository,
	}
}

// PlayerRank represents a player's statistics with ranking information
type PlayerRank struct {
	*entities.PlayerStatistics
	Rank       int     `json:"rank"`
	WinRate    float64 `json:"win_rate"`
	ProfitRate float64 `json:"profit_rate"`
	IsTopWinner bool   `json:"is_top_winner"`
	IsTopPlayer bool   `json:"is_top_player"`
}

// BlackjackLeaderboard represents a paginated leaderboard of player statistics
type BlackjackLeaderboard struct {
	Players       []*PlayerRank `json:"players"`
	TotalPlayers  int          `json:"total_players"`
	CurrentPage   int          `json:"current_page"`
	TotalPages    int          `json:"total_pages"`
	PlayersPerPage int         `json:"players_per_page"`
	LastUpdated   time.Time    `json:"last_updated"`
}

// GetBlackjackLeaderboard retrieves a paginated leaderboard of blackjack player statistics
func (s *Service) GetBlackjackLeaderboard(ctx context.Context, page, playersPerPage int) (*BlackjackLeaderboard, error) {
	// Default values
	if page < 1 {
		page = 1
	}
	if playersPerPage < 1 {
		playersPerPage = 10
	}

	// Get all player statistics for blackjack
	allStats, err := s.repository.GetAllPlayerStatistics(ctx, "blackjack")
	if err != nil {
		return nil, err
	}

	// Convert to PlayerRank and calculate additional metrics
	playerRanks := make([]*PlayerRank, 0, len(allStats))
	for _, stats := range allStats {
		// Skip players with no games
		if stats.GamesPlayed == 0 {
			continue
		}

		// Calculate win rate and profit rate
		winRate := float64(stats.Wins) / float64(stats.GamesPlayed)
		var profitRate float64
		if stats.TotalBet > 0 {
			profitRate = float64(stats.TotalWinnings) / float64(stats.TotalBet)
		}

		playerRanks = append(playerRanks, &PlayerRank{
			PlayerStatistics: stats,
			WinRate:         winRate,
			ProfitRate:      profitRate,
		})
	}

	// Sort by total winnings (descending)
	sort.Slice(playerRanks, func(i, j int) bool {
		return playerRanks[i].TotalWinnings > playerRanks[j].TotalWinnings
	})

	// Mark top winners and players
	if len(playerRanks) > 0 {
		// Top winner is the player with the highest winnings
		playerRanks[0].IsTopWinner = true

		// Find the player with the most games played
		mostGamesIdx := 0
		for i := 1; i < len(playerRanks); i++ {
			if playerRanks[i].GamesPlayed > playerRanks[mostGamesIdx].GamesPlayed {
				mostGamesIdx = i
			}
		}
		playerRanks[mostGamesIdx].IsTopPlayer = true
	}

	// Assign ranks
	for i := range playerRanks {
		playerRanks[i].Rank = i + 1
	}

	// Calculate pagination
	totalPlayers := len(playerRanks)
	totalPages := (totalPlayers + playersPerPage - 1) / playersPerPage
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	// Get the current page of players
	start := (page - 1) * playersPerPage
	end := start + playersPerPage
	if end > totalPlayers {
		end = totalPlayers
	}

	var currentPagePlayers []*PlayerRank
	if start < totalPlayers {
		currentPagePlayers = playerRanks[start:end]
	} else {
		currentPagePlayers = []*PlayerRank{}
	}

	// Create the leaderboard
	return &BlackjackLeaderboard{
		Players:        currentPagePlayers,
		TotalPlayers:   totalPlayers,
		CurrentPage:    page,
		TotalPages:     totalPages,
		PlayersPerPage: playersPerPage,
		LastUpdated:    time.Now(),
	}, nil
}

// EnsureStatisticsUpdated ensures that player statistics are updated for a completed game
func (s *Service) EnsureStatisticsUpdated(ctx context.Context, gameResult *entities.GameResult) error {
	// Update player statistics in the repository
	return s.repository.UpdatePlayerStatistics(ctx, gameResult)
}
