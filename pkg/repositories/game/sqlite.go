package game

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fadedpez/tucoramirez/pkg/db/migrations"
	"github.com/fadedpez/tucoramirez/pkg/entities"
	_ "github.com/mattn/go-sqlite3"
)

// SQLite table schemas
const (
	createDeckTableSQL = `
	CREATE TABLE IF NOT EXISTS decks (
		channel_id TEXT PRIMARY KEY,
		cards TEXT NOT NULL,  -- JSON array of cards
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	createGameResultsTableSQL = `
	CREATE TABLE IF NOT EXISTS game_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id TEXT NOT NULL,
		game_type TEXT NOT NULL,
		completed_at TIMESTAMP NOT NULL,
		details TEXT,  -- JSON for game-specific details
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_channel ON game_results(channel_id)`

	createPlayerResultsTableSQL = `
	CREATE TABLE IF NOT EXISTS player_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		game_result_id INTEGER NOT NULL,
		player_id TEXT NOT NULL,
		result TEXT NOT NULL,
		score INTEGER NOT NULL,
		FOREIGN KEY (game_result_id) REFERENCES game_results(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_player ON player_results(player_id)`

	createGameRecordsTableSQL = `
	CREATE TABLE IF NOT EXISTS game_records (
		id TEXT PRIMARY KEY,
		game_type TEXT NOT NULL,
		channel_id TEXT NOT NULL,
		start_time TIMESTAMP NOT NULL,
		end_time TIMESTAMP NOT NULL,
		dealer_cards TEXT NOT NULL,  -- JSON array of card strings
		dealer_score INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_game_records_channel ON game_records(channel_id)`

	createHandRecordsTableSQL = `
	CREATE TABLE IF NOT EXISTS hand_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		game_record_id TEXT NOT NULL,
		player_id TEXT NOT NULL,
		hand_id TEXT NOT NULL,
		parent_hand_id TEXT,
		cards TEXT NOT NULL,  -- JSON array of card strings
		final_score INTEGER NOT NULL,
		initial_bet INTEGER NOT NULL,
		is_split BOOLEAN NOT NULL,
		is_doubled_down BOOLEAN NOT NULL,
		double_down_bet INTEGER,
		has_insurance BOOLEAN NOT NULL,
		insurance_bet INTEGER,
		result TEXT NOT NULL,
		payout INTEGER NOT NULL,
		insurance_payout INTEGER,
		actions TEXT NOT NULL,  -- JSON array of action strings
		metadata TEXT,  -- JSON object for additional metadata
		FOREIGN KEY (game_record_id) REFERENCES game_records(id) ON DELETE CASCADE
	);
	CREATE INDEX IF NOT EXISTS idx_hand_records_player ON hand_records(player_id);
	CREATE INDEX IF NOT EXISTS idx_hand_records_game ON hand_records(game_record_id)`
)

// SQLiteRepository implements the Repository interface using SQLite
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(dbPath string) (*SQLiteRepository, error) {
	// Ensure the directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("error creating database directory: %w", err)
	}

	// Open the database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Apply migrations
	migrator := migrations.NewMigrator(db, "migrations")
	if err := migrator.MigrateUp(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error applying migrations: %w", err)
	}

	return &SQLiteRepository{db: db}, nil
}

// SaveDeck stores a deck for a channel
func (r *SQLiteRepository) SaveDeck(ctx context.Context, channelID string, deck []*entities.Card) error {
	// Convert deck to JSON
	cardsJSON, err := json.Marshal(deck)
	if err != nil {
		return err
	}

	// Use UPSERT syntax for SQLite
	query := `
		INSERT INTO decks (channel_id, cards, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(channel_id) 
		DO UPDATE SET cards = ?, updated_at = CURRENT_TIMESTAMP`

	_, err = r.db.ExecContext(ctx, query, channelID, cardsJSON, cardsJSON)
	return err
}

// GetDeck retrieves a deck for a channel
func (r *SQLiteRepository) GetDeck(ctx context.Context, channelID string) ([]*entities.Card, error) {
	var cardsJSON []byte
	query := `SELECT cards FROM decks WHERE channel_id = ?`

	err := r.db.QueryRowContext(ctx, query, channelID).Scan(&cardsJSON)
	if err == sql.ErrNoRows {
		return nil, nil // Return empty deck if none exists
	}
	if err != nil {
		return nil, err
	}

	var deck []*entities.Card
	if err := json.Unmarshal(cardsJSON, &deck); err != nil {
		return nil, err
	}

	return deck, nil
}

// SaveGameResult stores a game result
func (r *SQLiteRepository) SaveGameResult(ctx context.Context, result *entities.GameResult) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Serialize game details if present
	var detailsJSON []byte
	if result.Details != nil {
		detailsJSON, err = json.Marshal(result.Details)
		if err != nil {
			return err
		}
	}

	// Insert game result
	query := `
		INSERT INTO game_results (
			channel_id, game_type, completed_at, details
		) VALUES (?, ?, ?, ?)`

	res, err := tx.ExecContext(ctx, query,
		result.ChannelID, result.GameType, result.CompletedAt, detailsJSON)
	if err != nil {
		return err
	}

	// Get the ID of the inserted game result
	gameResultID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	// Insert player results
	for _, pr := range result.PlayerResults {
		query := `
			INSERT INTO player_results (
				game_result_id, player_id, result, score
			) VALUES (?, ?, ?, ?)`

		_, err = tx.ExecContext(ctx, query,
			gameResultID, pr.PlayerID, pr.Result, pr.Score)
		if err != nil {
			return err
		}
	}

	// Commit transaction
	return tx.Commit()
}

// GetPlayerResults retrieves game results for a player
func (r *SQLiteRepository) GetPlayerResults(ctx context.Context, playerID string) ([]*entities.GameResult, error) {
	query := `
		SELECT gr.id, gr.channel_id, gr.game_type, gr.completed_at, gr.details,
			   pr.player_id, pr.result, pr.score
		FROM game_results gr
		JOIN player_results pr ON gr.id = pr.game_result_id
		WHERE pr.player_id = ?
		ORDER BY gr.completed_at DESC`

	rows, err := r.db.QueryContext(ctx, query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*entities.GameResult
	resultMap := make(map[int64]*entities.GameResult)

	for rows.Next() {
		var (
			gameID      int64
			channelID   string
			gameType    string
			completedAt time.Time
			detailsJSON []byte
			playerID    string
			resultStr   string
			score       int
		)

		err := rows.Scan(
			&gameID, &channelID, &gameType, &completedAt, &detailsJSON,
			&playerID, &resultStr, &score,
		)
		if err != nil {
			return nil, err
		}

		// Check if we already have this game result
		result, exists := resultMap[gameID]
		if !exists {
			// Create new game result
			result = &entities.GameResult{
				ChannelID:     channelID,
				GameType:      entities.GameState(gameType),
				CompletedAt:   completedAt,
				PlayerResults: []*entities.PlayerResult{},
			}

			// Unmarshal details if present
			if len(detailsJSON) > 0 {
				// Note: We can't unmarshal directly to GameDetails interface
				// In a real implementation, you'd need type information to unmarshal correctly
				// This is a placeholder for how you might handle it
			}

			resultMap[gameID] = result
			results = append(results, result)
		}

		// Add player result
		playerResult := &entities.PlayerResult{
			PlayerID: playerID,
			Result:   entities.StringResult(resultStr),
			Score:    score,
		}
		result.PlayerResults = append(result.PlayerResults, playerResult)
	}

	return results, nil
}

// GetChannelResults retrieves recent game results for a channel
func (r *SQLiteRepository) GetChannelResults(ctx context.Context, channelID string, limit int) ([]*entities.GameResult, error) {
	query := `
		SELECT gr.id, gr.channel_id, gr.game_type, gr.completed_at, gr.details
		FROM game_results gr
		WHERE gr.channel_id = ?
		ORDER BY gr.completed_at DESC
		LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, channelID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gameIDs []int64
	var results []*entities.GameResult
	resultMap := make(map[int64]*entities.GameResult)

	// First pass: get all game results
	for rows.Next() {
		var (
			gameID      int64
			channelID   string
			gameType    string
			completedAt time.Time
			detailsJSON []byte
		)

		err := rows.Scan(
			&gameID, &channelID, &gameType, &completedAt, &detailsJSON,
		)
		if err != nil {
			return nil, err
		}

		// Create new game result
		result := &entities.GameResult{
			ChannelID:     channelID,
			GameType:      entities.GameState(gameType),
			CompletedAt:   completedAt,
			PlayerResults: []*entities.PlayerResult{},
		}

		// Unmarshal details if present
		if len(detailsJSON) > 0 {
			// Note: We can't unmarshal directly to GameDetails interface
			// In a real implementation, you'd need type information to unmarshal correctly
		}

		resultMap[gameID] = result
		results = append(results, result)
		gameIDs = append(gameIDs, gameID)
	}

	// If no results, return empty slice
	if len(gameIDs) == 0 {
		return []*entities.GameResult{}, nil
	}

	// Second pass: get player results for each game
	placeholders := make([]string, len(gameIDs))
	args := make([]interface{}, len(gameIDs))
	for i, id := range gameIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query = `
		SELECT game_result_id, player_id, result, score
		FROM player_results
		WHERE game_result_id IN (` +
		// Join placeholders with commas
		func() string {
			result := ""
			for i, p := range placeholders {
				if i > 0 {
					result += ","
				}
				result += p
			}
			return result
		}() + `)`

	rows, err = r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Add player results to their respective games
	for rows.Next() {
		var (
			gameID    int64
			playerID  string
			resultStr string
			score     int
		)

		err := rows.Scan(&gameID, &playerID, &resultStr, &score)
		if err != nil {
			return nil, err
		}

		// Find the game result and add the player result
		if result, exists := resultMap[gameID]; exists {
			playerResult := &entities.PlayerResult{
				PlayerID: playerID,
				Result:   entities.StringResult(resultStr),
				Score:    score,
			}
			result.PlayerResults = append(result.PlayerResults, playerResult)
		}
	}

	return results, nil
}

// Close closes the database connection
func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}
