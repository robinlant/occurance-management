CREATE TABLE IF NOT EXISTS comments (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    occurrence_id INTEGER NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    user_id       INTEGER NOT NULL REFERENCES users(id)       ON DELETE CASCADE,
    body          TEXT    NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_comments_occurrence ON comments(occurrence_id);
