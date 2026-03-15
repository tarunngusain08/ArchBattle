CREATE TABLE IF NOT EXISTS discussion_upvotes (
    entry_id UUID NOT NULL REFERENCES discussion_entries(id) ON DELETE CASCADE,
    voter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, voter_id)
);
