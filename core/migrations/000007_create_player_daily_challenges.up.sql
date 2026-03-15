CREATE TABLE IF NOT EXISTS player_daily_challenges (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    challenge_date DATE NOT NULL REFERENCES daily_challenges(challenge_date) ON DELETE CASCADE,
    score INTEGER NOT NULL DEFAULT 0,
    streak_day INTEGER NOT NULL DEFAULT 0,
    share_card_text TEXT NOT NULL DEFAULT '',
    completed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, challenge_date)
);
