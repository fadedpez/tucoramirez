-- Migration: add_player_statistics_table.sql

-- Create player_statistics table
CREATE TABLE IF NOT EXISTS player_statistics (
    player_id TEXT NOT NULL,
    game_type TEXT NOT NULL,
    games_played INTEGER DEFAULT 0,
    wins INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    pushes INTEGER DEFAULT 0,
    blackjacks INTEGER DEFAULT 0,
    busts INTEGER DEFAULT 0,
    splits INTEGER DEFAULT 0,
    double_downs INTEGER DEFAULT 0,
    insurances INTEGER DEFAULT 0,
    total_bet INTEGER DEFAULT 0,
    total_winnings INTEGER DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (player_id, game_type)
);

-- Create index for faster queries
CREATE INDEX IF NOT EXISTS idx_player_statistics_game_type ON player_statistics(game_type);
