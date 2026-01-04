-- Create sets table to store Pokemon TCG set metadata
CREATE TABLE IF NOT EXISTS sets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_id TEXT UNIQUE NOT NULL,
    code TEXT,
    name TEXT NOT NULL,
    series TEXT,
    printed_total INTEGER,
    total INTEGER,
    release_date TEXT,
    symbol_url TEXT,
    logo_url TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sets_api_id ON sets(api_id);
CREATE INDEX IF NOT EXISTS idx_sets_release_date ON sets(release_date);
