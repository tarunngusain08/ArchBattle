CREATE TABLE IF NOT EXISTS daily_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_date DATE NOT NULL UNIQUE,
    question_ids UUID[] NOT NULL DEFAULT '{}',
    theme TEXT NOT NULL,
    ai_summary TEXT NOT NULL DEFAULT '',
    summary_reviewed BOOLEAN NOT NULL DEFAULT FALSE,
    expert_editor UUID REFERENCES users(id),
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
