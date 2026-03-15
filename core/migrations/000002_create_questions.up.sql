CREATE TABLE IF NOT EXISTS questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mode TEXT NOT NULL,
    topic TEXT NOT NULL,
    difficulty_tier TEXT NOT NULL CHECK (difficulty_tier IN ('junior', 'senior', 'staff')),
    prompt TEXT NOT NULL,
    options JSONB NOT NULL,
    correct_answers JSONB NOT NULL,
    rationale TEXT NOT NULL,
    dispute_count INTEGER NOT NULL DEFAULT 0,
    pilot_attempts INTEGER NOT NULL DEFAULT 0,
    pilot_dispute_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    daily_eligible BOOLEAN NOT NULL DEFAULT FALSE,
    reviewed_by UUID REFERENCES users(id),
    second_reviewer UUID REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'staged', 'pilot', 'live', 'retired', 'quarantined')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
