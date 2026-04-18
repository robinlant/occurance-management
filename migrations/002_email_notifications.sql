CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS email_notifications_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_type TEXT    NOT NULL,
    sent_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_email_log_user_sent ON email_notifications_log(user_id, sent_at);

-- Default settings
INSERT OR IGNORE INTO settings (key, value) VALUES ('email_enabled', 'false');
INSERT OR IGNORE INTO settings (key, value) VALUES ('smtp_host', '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('smtp_port', '587');
INSERT OR IGNORE INTO settings (key, value) VALUES ('smtp_username', '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('smtp_password', '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('sender_email', '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('sender_name', 'DutyRound');
INSERT OR IGNORE INTO settings (key, value) VALUES ('max_emails_per_day', '1');
INSERT OR IGNORE INTO settings (key, value) VALUES ('upcoming_reminder_days', '3');
