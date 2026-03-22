-- Bulls and Cows Database Schema
-- Creates tables for game state and guess history

-- Games table
CREATE TABLE IF NOT EXISTS games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mode VARCHAR(20) NOT NULL CHECK (mode IN ('colors', 'numbers')),
    variant VARCHAR(20) NOT NULL CHECK (variant IN ('1player', '2player')),
    secret_code VARCHAR(10) NOT NULL,  -- e.g., "RBGY" or "1234"
    code_maker VARCHAR(255) NOT NULL,  -- user_id or "AI"
    code_breaker VARCHAR(255) NOT NULL,  -- user_id
    max_guesses INTEGER DEFAULT 12,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'won', 'lost', 'abandoned')),
    winner VARCHAR(255),  -- user_id or NULL
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_games_code_breaker ON games(code_breaker);
CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
CREATE INDEX IF NOT EXISTS idx_games_created ON games(created_at DESC);

-- Guesses table
CREATE TABLE IF NOT EXISTS guesses (
    id SERIAL PRIMARY KEY,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    guess_number INTEGER NOT NULL,  -- 1-12
    guess_code VARCHAR(10) NOT NULL,  -- e.g., "RBGY" or "1234"
    bulls INTEGER NOT NULL,  -- 0-4
    cows INTEGER NOT NULL,   -- 0-4
    guessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_guesses_game ON guesses(game_id, guess_number);

-- Auto-update trigger for games.updated_at
CREATE OR REPLACE FUNCTION update_games_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS games_updated_at ON games;
CREATE TRIGGER games_updated_at
    BEFORE UPDATE ON games
    FOR EACH ROW
    EXECUTE FUNCTION update_games_updated_at();
