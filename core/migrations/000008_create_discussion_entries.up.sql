CREATE TABLE IF NOT EXISTS discussion_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_date DATE NOT NULL REFERENCES daily_challenges(challenge_date) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    question_number INTEGER NOT NULL,
    reasoning_text TEXT NOT NULL DEFAULT '',
    alternative_text TEXT NOT NULL DEFAULT '',
    surprise_text TEXT NOT NULL DEFAULT '',
    upvotes INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
