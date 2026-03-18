CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT    NOT NULL,
    email         TEXT    NOT NULL UNIQUE,
    role          TEXT    NOT NULL CHECK (role IN ('admin', 'organizer', 'participant')),
    password_hash TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS groups (
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS occurrences (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id         INTEGER REFERENCES groups(id) ON DELETE SET NULL,
    title            TEXT     NOT NULL,
    description      TEXT     NOT NULL DEFAULT '',
    date             DATETIME NOT NULL,
    min_participants INTEGER  NOT NULL DEFAULT 1,
    max_participants INTEGER  NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS participations (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    occurrence_id INTEGER NOT NULL REFERENCES occurrences(id) ON DELETE CASCADE,
    UNIQUE (user_id, occurrence_id)
);

CREATE TABLE IF NOT EXISTS out_of_office (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id   INTEGER  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    from_date DATETIME NOT NULL,
    to_date   DATETIME NOT NULL
);
