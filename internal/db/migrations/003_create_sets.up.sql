CREATE TABLE "sets" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    api_id TEXT UNIQUE NOT NULL,
    code TEXT,
    name TEXT NOT NULL,
    printed_total INTEGER,
    total INTEGER,
    symbol_url TEXT,
    logo_url TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sets_api_id ON sets(api_id);