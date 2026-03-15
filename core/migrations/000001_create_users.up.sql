CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    tier TEXT NOT NULL DEFAULT 'junior' CHECK (tier IN ('junior', 'senior', 'staff')),
    junior_elo INTEGER NOT NULL DEFAULT 1000,
    senior_elo INTEGER NOT NULL DEFAULT 1000,
    staff_elo INTEGER NOT NULL DEFAULT 1000,
    matches_played INTEGER NOT NULL DEFAULT 0,
    current_streak INTEGER NOT NULL DEFAULT 0,
    longest_streak INTEGER NOT NULL DEFAULT 0,
    last_daily_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
