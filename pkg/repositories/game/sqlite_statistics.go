package game

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/entities"
)

// GetPlayerStatistics retrieves statistics for a specific player and game type
func (r *SQLiteRepository) GetPlayerStatistics(ctx context.Context, playerID string, gameType entities.GameState) (*entities.PlayerStatistics, error) {
	query := `
		SELECT player_id, game_type, games_played, wins, losses, pushes, 
		       blackjacks, busts, splits, double_downs, insurances,
		       total_bet, total_winnings, last_updated
		FROM player_statistics
		WHERE player_id = ? AND game_type = ?
	`
	
	var stats entities.PlayerStatistics
	err := r.db.QueryRowContext(ctx, query, playerID, string(gameType)).Scan(
		&stats.PlayerID, &stats.GameType, &stats.GamesPlayed, &stats.Wins, 
		&stats.Losses, &stats.Pushes, &stats.Blackjacks, &stats.Busts, 
		&stats.Splits, &stats.DoubleDowns, &stats.Insurances,
		&stats.TotalBet, &stats.TotalWinnings, &stats.LastUpdated,
	)
	
	if err == sql.ErrNoRows {
		// Return empty statistics if not found
		return &entities.PlayerStatistics{
			PlayerID: playerID,
			GameType: gameType,
		}, nil
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get player statistics: %w", err)
	}
	
	return &stats, nil
}

// GetAllPlayerStatistics retrieves statistics for all players for a specific game type
func (r *SQLiteRepository) GetAllPlayerStatistics(ctx context.Context, gameType entities.GameState) ([]*entities.PlayerStatistics, error) {
	query := `
		SELECT player_id, game_type, games_played, wins, losses, pushes, 
		       blackjacks, busts, splits, double_downs, insurances,
		       total_bet, total_winnings, last_updated
		FROM player_statistics
		WHERE game_type = ?
		ORDER BY total_winnings DESC
	`
	
	rows, err := r.db.QueryContext(ctx, query, string(gameType))
	if err != nil {
		return nil, fmt.Errorf("failed to query player statistics: %w", err)
	}
	defer rows.Close()
	
	var statsList []*entities.PlayerStatistics
	for rows.Next() {
		var stats entities.PlayerStatistics
		if err := rows.Scan(
			&stats.PlayerID, &stats.GameType, &stats.GamesPlayed, &stats.Wins, 
			&stats.Losses, &stats.Pushes, &stats.Blackjacks, &stats.Busts, 
			&stats.Splits, &stats.DoubleDowns, &stats.Insurances,
			&stats.TotalBet, &stats.TotalWinnings, &stats.LastUpdated,
		); err != nil {
			return nil, fmt.Errorf("failed to scan player statistics: %w", err)
		}
		statsList = append(statsList, &stats)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating player statistics: %w", err)
	}
	
	return statsList, nil
}

// UpdatePlayerStatistics updates statistics for all players in a game result
func (r *SQLiteRepository) UpdatePlayerStatistics(ctx context.Context, gameResult *entities.GameResult) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Process each player's result
	for _, playerResult := range gameResult.PlayerResults {
		// Skip split hands that are tracked with the parent hand if that info is in metadata
		if playerResult.Metadata != nil {
			if isSplitHand, ok := playerResult.Metadata["is_split_hand"].(bool); ok && isSplitHand {
				continue
			}
		}
		
		// Get current statistics
		var stats entities.PlayerStatistics
		query := `
			SELECT player_id, game_type, games_played, wins, losses, pushes, 
			       blackjacks, busts, splits, double_downs, insurances,
			       total_bet, total_winnings, last_updated
			FROM player_statistics
			WHERE player_id = ? AND game_type = ?
		`
		
		err := tx.QueryRowContext(ctx, query, playerResult.PlayerID, string(gameResult.GameType)).Scan(
			&stats.PlayerID, &stats.GameType, &stats.GamesPlayed, &stats.Wins, 
			&stats.Losses, &stats.Pushes, &stats.Blackjacks, &stats.Busts, 
			&stats.Splits, &stats.DoubleDowns, &stats.Insurances,
			&stats.TotalBet, &stats.TotalWinnings, &stats.LastUpdated,
		)
		
		// If no statistics exist yet, create a new record
		if err == sql.ErrNoRows {
			stats = entities.PlayerStatistics{
				PlayerID:    playerResult.PlayerID,
				GameType:    gameResult.GameType,
				GamesPlayed: 0,
			}
		} else if err != nil {
			return fmt.Errorf("failed to get player statistics: %w", err)
		}
		
		// Update statistics based on game result
		stats.GamesPlayed++
		stats.TotalBet += playerResult.Bet
		stats.TotalWinnings += playerResult.Payout
		
		// Update result counters
		switch playerResult.Result.String() {
		case entities.StringResultWin.String():
			stats.Wins++
		case entities.StringResultLose.String():
			stats.Losses++
		case entities.StringResultPush.String():
			stats.Pushes++
		}
		
		// Update special stats for blackjack
		if gameResult.GameType == entities.StateComplete {
			// Check for blackjack
			if playerResult.Result.String() == entities.StringResultBlackjack.String() {
				stats.Blackjacks++
			}
			
			// Check for bust
			if playerResult.Metadata != nil {
				if busted, ok := playerResult.Metadata["busted"].(bool); ok && busted {
					stats.Busts++
				}
				
				// Check for split
				if hasSplit, ok := playerResult.Metadata["split"].(bool); ok && hasSplit {
					stats.Splits++
				}
				
				// Check for double down
				if isDoubledDown, ok := playerResult.Metadata["doubled_down"].(bool); ok && isDoubledDown {
					stats.DoubleDowns++
				}
				
				// Check for insurance
				if hasInsurance, ok := playerResult.Metadata["insurance"].(bool); ok && hasInsurance {
					stats.Insurances++
				}
			}
		}
		
		// Update or insert the statistics
		if err == sql.ErrNoRows {
			// Insert new record
			insertQuery := `
				INSERT INTO player_statistics (
					player_id, game_type, games_played, wins, losses, pushes,
					blackjacks, busts, splits, double_downs, insurances,
					total_bet, total_winnings, last_updated
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`
			
			_, err = tx.ExecContext(ctx, insertQuery,
				stats.PlayerID, string(stats.GameType), stats.GamesPlayed, stats.Wins,
				stats.Losses, stats.Pushes, stats.Blackjacks, stats.Busts,
				stats.Splits, stats.DoubleDowns, stats.Insurances,
				stats.TotalBet, stats.TotalWinnings, time.Now(),
			)
		} else {
			// Update existing record
			updateQuery := `
				UPDATE player_statistics SET
					games_played = ?,
					wins = ?,
					losses = ?,
					pushes = ?,
					blackjacks = ?,
					busts = ?,
					splits = ?,
					double_downs = ?,
					insurances = ?,
					total_bet = ?,
					total_winnings = ?,
					last_updated = ?
				WHERE player_id = ? AND game_type = ?
			`
			
			_, err = tx.ExecContext(ctx, updateQuery,
				stats.GamesPlayed, stats.Wins, stats.Losses, stats.Pushes,
				stats.Blackjacks, stats.Busts, stats.Splits, stats.DoubleDowns, stats.Insurances,
				stats.TotalBet, stats.TotalWinnings, time.Now(),
				stats.PlayerID, string(stats.GameType),
			)
		}
		
		if err != nil {
			return fmt.Errorf("failed to update player statistics: %w", err)
		}
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// PruneGameResultsPerPlayer removes old game results, keeping only the specified number of recent matches per player
func (r *SQLiteRepository) PruneGameResultsPerPlayer(ctx context.Context, maxMatchesPerPlayer int) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get all player IDs
	rows, err := tx.QueryContext(ctx, "SELECT DISTINCT player_id FROM player_results")
	if err != nil {
		return fmt.Errorf("failed to query player IDs: %w", err)
	}
	defer rows.Close()

	// For each player, keep only the most recent matches
	for rows.Next() {
		var playerID string
		if err := rows.Scan(&playerID); err != nil {
			return fmt.Errorf("failed to scan player ID: %w", err)
		}

		// Get game result IDs to keep for this player
		keepRows, err := tx.QueryContext(ctx, `
			SELECT pr.game_result_id
			FROM player_results pr
			JOIN game_results gr ON pr.game_result_id = gr.id
			WHERE pr.player_id = ?
			ORDER BY gr.completed_at DESC
			LIMIT ?
		`, playerID, maxMatchesPerPlayer)
		if err != nil {
			return fmt.Errorf("failed to query game results to keep: %w", err)
		}

		var keepIDs []int64
		for keepRows.Next() {
			var id int64
			if err := keepRows.Scan(&id); err != nil {
				keepRows.Close()
				return fmt.Errorf("failed to scan game result ID: %w", err)
			}
			keepIDs = append(keepIDs, id)
		}
		keepRows.Close()

		if err := keepRows.Err(); err != nil {
			return fmt.Errorf("error iterating game result IDs: %w", err)
		}

		// If no results to keep, continue to the next player
		if len(keepIDs) == 0 {
			continue
		}

		// Get game result IDs to delete for this player
		deleteRows, err := tx.QueryContext(ctx, `
			SELECT pr.game_result_id
			FROM player_results pr
			JOIN game_results gr ON pr.game_result_id = gr.id
			WHERE pr.player_id = ?
			ORDER BY gr.completed_at DESC
			LIMIT -1 OFFSET ?
		`, playerID, maxMatchesPerPlayer)
		if err != nil {
			return fmt.Errorf("failed to query game results to delete: %w", err)
		}

		var deleteIDs []int64
		for deleteRows.Next() {
			var id int64
			if err := deleteRows.Scan(&id); err != nil {
				deleteRows.Close()
				return fmt.Errorf("failed to scan game result ID: %w", err)
			}
			deleteIDs = append(deleteIDs, id)
		}
		deleteRows.Close()

		if err := deleteRows.Err(); err != nil {
			return fmt.Errorf("error iterating game result IDs: %w", err)
		}

		// If no results to delete, continue to the next player
		if len(deleteIDs) == 0 {
			continue
		}

		// Delete player results for the game results to delete
		for _, id := range deleteIDs {
			_, err := tx.ExecContext(ctx, "DELETE FROM player_results WHERE game_result_id = ? AND player_id = ?", id, playerID)
			if err != nil {
				return fmt.Errorf("failed to delete player results: %w", err)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ArchiveGameResults archives game results to JSON files
func (r *SQLiteRepository) ArchiveGameResults(ctx context.Context, gameResults []*entities.GameResult) error {
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

// IndexGameResult is a no-op for SQLiteRepository
func (r *SQLiteRepository) IndexGameResult(ctx context.Context, gameResult *entities.GameResult) error {
	// This is a no-op for SQLiteRepository, as it doesn't use Elasticsearch
	return nil
}

// GetPlayerStatisticsFromES is a no-op for SQLiteRepository
func (r *SQLiteRepository) GetPlayerStatisticsFromES(ctx context.Context, playerID string, gameType entities.GameState) (*entities.PlayerStatistics, error) {
	// This is a no-op for SQLiteRepository, as it doesn't use Elasticsearch
	// Just delegate to the regular GetPlayerStatistics method
	return r.GetPlayerStatistics(ctx, playerID, gameType)
}

// GetAllPlayerStatisticsFromES is a no-op for SQLiteRepository
func (r *SQLiteRepository) GetAllPlayerStatisticsFromES(ctx context.Context, gameType entities.GameState) ([]*entities.PlayerStatistics, error) {
	// This is a no-op for SQLiteRepository, as it doesn't use Elasticsearch
	// Just delegate to the regular GetAllPlayerStatistics method
	return r.GetAllPlayerStatistics(ctx, gameType)
}
