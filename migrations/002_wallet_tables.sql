-- Migration: Wallet tables
-- Created: 2025-03-13T16:54:00-04:00

-- Create wallets table
CREATE TABLE IF NOT EXISTS wallets (
    user_id TEXT PRIMARY KEY,
    balance INTEGER NOT NULL DEFAULT 100,
    loan_amount INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    amount INTEGER NOT NULL,
    type TEXT NOT NULL,
    reference_id TEXT,
    description TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    balance_after INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES wallets(user_id)
);

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_transactions_timestamp ON transactions(timestamp DESC);
