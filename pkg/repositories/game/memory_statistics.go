package game

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

// Add statistics fields to MemoryRepository
func init() {
	// This is just a placeholder to indicate that we're extending the MemoryRepository
	// The actual fields are added to the struct in memory.go
}

// GetPlayerStatistics retrieves statistics for a specific player and game type
func (r *MemoryRepository) GetPlayerStatistics(ctx context.Context, playerID string, gameType entities.GameState) (*entities.PlayerStatistics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create empty statistics
	stats := &entities.PlayerStatistics{
		PlayerID:    playerID,
		GameType:    gameType,
		LastUpdated: time.Now(),
	}

	// Get player results
	results, ok := r.playerResults[playerID]
	if !ok || len(results) == 0 {
		return stats, nil
	}

	// Calculate statistics from results
	for _, result := range results {
		// Skip results that don't match the game type
		if result.GameType != gameType {
			continue
		}

		// Process each player result
		for _, pr := range result.PlayerResults {
			// Skip results that don't belong to this player
			if pr.PlayerID != playerID {
				continue
			}

			// Skip split hands that are tracked with the parent hand
			if pr.Metadata != nil {
				if isSplitHand, ok := pr.Metadata["is_split_hand"].(bool); ok && isSplitHand {
					continue
				}
			}

			// Update core statistics
			stats.GamesPlayed++
			stats.TotalBet += pr.Bet
			stats.TotalWinnings += pr.Payout

			// Update result counters
			switch pr.Result.String() {
			case entities.StringResultWin.String():
				stats.Wins++
			case entities.StringResultLose.String():
				stats.Losses++
			case entities.StringResultPush.String():
				stats.Pushes++
			}

			// Update special stats for blackjack
			if result.GameType == entities.StateComplete {
				// Check for blackjack
				if pr.Result.String() == entities.StringResultBlackjack.String() {
					stats.Blackjacks++
				}

				// Check for bust
				if pr.Metadata != nil {
					if busted, ok := pr.Metadata["busted"].(bool); ok && busted {
						stats.Busts++
					}

					// Check for split
					if hasSplit, ok := pr.Metadata["split"].(bool); ok && hasSplit {
						stats.Splits++
					}

					// Check for double down
					if isDoubledDown, ok := pr.Metadata["doubled_down"].(bool); ok && isDoubledDown {
						stats.DoubleDowns++
					}

					// Check for insurance
					if hasInsurance, ok := pr.Metadata["insurance"].(bool); ok && hasInsurance {
						stats.Insurances++
					}
				}
			}
		}
	}

	return stats, nil
}

// GetAllPlayerStatistics retrieves statistics for all players for a specific game type
func (r *MemoryRepository) GetAllPlayerStatistics(ctx context.Context, gameType entities.GameState) ([]*entities.PlayerStatistics, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get all player IDs
	playerIDs := make(map[string]bool)
	for playerID := range r.playerResults {
		playerIDs[playerID] = true
	}

	// Get statistics for each player
	var statsList []*entities.PlayerStatistics
	for playerID := range playerIDs {
		stats, err := r.GetPlayerStatistics(ctx, playerID, gameType)
		if err != nil {
			return nil, err
		}

		// Only include players who have played this game type
		if stats.GamesPlayed > 0 {
			statsList = append(statsList, stats)
		}
	}

	// Sort by total winnings (descending)
	// Note: In a real implementation, you'd want to use a proper sorting algorithm
	for i := 0; i < len(statsList); i++ {
		for j := i + 1; j < len(statsList); j++ {
			if statsList[i].TotalWinnings < statsList[j].TotalWinnings {
				statsList[i], statsList[j] = statsList[j], statsList[i]
			}
		}
	}

	return statsList, nil
}

// UpdatePlayerStatistics updates statistics for all players in a game result
func (r *MemoryRepository) UpdatePlayerStatistics(ctx context.Context, gameResult *entities.GameResult) error {
	// For the memory repository, we don't need to do anything here
	// since we calculate statistics on the fly from the game results
	return nil
}

// PruneGameResultsPerPlayer removes old game results, keeping only the specified number of recent matches per player
func (r *MemoryRepository) PruneGameResultsPerPlayer(ctx context.Context, maxMatchesPerPlayer int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Prune player results
	for playerID, results := range r.playerResults {
		if len(results) > maxMatchesPerPlayer {
			// Sort results by completion time (newest first)
			for i := 0; i < len(results); i++ {
				for j := i + 1; j < len(results); j++ {
					if results[i].CompletedAt.Before(results[j].CompletedAt) {
						results[i], results[j] = results[j], results[i]
					}
				}
			}

			// Keep only the most recent matches
			r.playerResults[playerID] = results[:maxMatchesPerPlayer]
		}
	}

	return nil
}

// ArchiveGameResults archives game results to JSON files
func (r *MemoryRepository) ArchiveGameResults(ctx context.Context, gameResults []*entities.GameResult) error {
	// Create archive directory if it doesn't exist
	archiveDir := "./archives"
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Archive each game result
	for _, result := range gameResults {
		// Generate a unique filename
		filename := fmt.Sprintf(
			"%s_%s_%s.json",
				result.ChannelID,
				string(result.GameType),
				result.CompletedAt.Format("20060102_150405"),
		)
		filePath := filepath.Join(archiveDir, filename)

		// Marshal the game result to JSON
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal game result: %w", err)
		}

		// Write to file
		if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
			return fmt.Errorf("failed to write archive file: %w", err)
		}
	}

	return nil
}

// IndexGameResult is a no-op for MemoryRepository
func (r *MemoryRepository) IndexGameResult(ctx context.Context, gameResult *entities.GameResult) error {
	// This is a no-op for MemoryRepository, as it doesn't use Elasticsearch
	return nil
}

// GetPlayerStatisticsFromES is a no-op for MemoryRepository
func (r *MemoryRepository) GetPlayerStatisticsFromES(ctx context.Context, playerID string, gameType entities.GameState) (*entities.PlayerStatistics, error) {
	// This is a no-op for MemoryRepository, as it doesn't use Elasticsearch
	// Just delegate to the regular GetPlayerStatistics method
	return r.GetPlayerStatistics(ctx, playerID, gameType)
}

// GetAllPlayerStatisticsFromES is a no-op for MemoryRepository
func (r *MemoryRepository) GetAllPlayerStatisticsFromES(ctx context.Context, gameType entities.GameState) ([]*entities.PlayerStatistics, error) {
	// This is a no-op for MemoryRepository, as it doesn't use Elasticsearch
	// Just delegate to the regular GetAllPlayerStatistics method
	return r.GetAllPlayerStatistics(ctx, gameType)
}
