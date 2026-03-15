CREATE TABLE IF NOT EXISTS matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mode TEXT NOT NULL,
    topic TEXT NOT NULL,
    tier TEXT NOT NULL CHECK (tier IN ('junior', 'senior', 'staff')),
    status TEXT NOT NULL CHECK (status IN ('created', 'lobby', 'active', 'revealing', 'scoring', 'ended', 'abandoned')),
    question_ids UUID[] NOT NULL DEFAULT '{}',
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
