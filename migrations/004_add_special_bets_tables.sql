-- Migration: add special bets tables
-- Created: 2025-04-08T21:42:39-04:00


-- SQLite Examples:

-- Create a new table
-- CREATE TABLE IF NOT EXISTS table_name (
--   id INTEGER PRIMARY KEY AUTOINCREMENT,
--   name TEXT NOT NULL,
--   value INTEGER DEFAULT 0,
--   created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );

-- Add a column to existing table
-- ALTER TABLE table_name ADD COLUMN new_column TEXT;

-- Create an index
-- CREATE INDEX IF NOT EXISTS idx_table_column ON table_name(column_name);

-- Your migration SQL goes below this line:

-- Add columns to game_results table for special bets
ALTER TABLE game_results ADD COLUMN has_split INTEGER DEFAULT 0;
ALTER TABLE game_results ADD COLUMN is_doubled_down INTEGER DEFAULT 0;
ALTER TABLE game_results ADD COLUMN double_down_bet INTEGER DEFAULT 0;
ALTER TABLE game_results ADD COLUMN has_insurance INTEGER DEFAULT 0;
ALTER TABLE game_results ADD COLUMN insurance_bet INTEGER DEFAULT 0;
ALTER TABLE game_results ADD COLUMN insurance_payout INTEGER DEFAULT 0;
ALTER TABLE game_results ADD COLUMN parent_hand_id TEXT;
