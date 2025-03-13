-- Migration: Initial schema
-- Created: 2025-03-12T19:30:00-04:00

-- Create decks table
CREATE TABLE IF NOT EXISTS decks (
    channel_id TEXT PRIMARY KEY,
    cards TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create game_results table
CREATE TABLE IF NOT EXISTS game_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    channel_id TEXT NOT NULL,
    game_type TEXT NOT NULL,
    completed_at TIMESTAMP NOT NULL,
    details TEXT NOT NULL
);

-- Create player_results table
CREATE TABLE IF NOT EXISTS player_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_result_id INTEGER NOT NULL,
    player_id TEXT NOT NULL,
    result TEXT NOT NULL,
    score INTEGER NOT NULL,
    FOREIGN KEY (game_result_id) REFERENCES game_results(id) ON DELETE CASCADE
);

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_game_results_channel_id ON game_results(channel_id);
CREATE INDEX IF NOT EXISTS idx_player_results_player_id ON player_results(player_id);
CREATE INDEX IF NOT EXISTS idx_player_results_game_result_id ON player_results(game_result_id);
