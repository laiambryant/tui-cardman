-- Remove series and release_date from sets table
DROP INDEX IF EXISTS idx_sets_release_date;

-- Create a new table without the unwanted columns
CREATE TABLE IF NOT EXISTS sets_new (
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

-- Copy data from old table to new table
INSERT INTO sets_new (id, api_id, code, name, printed_total, total, symbol_url, logo_url, updated_at)
SELECT id, api_id, code, name, printed_total, total, symbol_url, logo_url, updated_at
FROM sets;

-- Drop the old table
DROP TABLE sets;

-- Rename the new table
ALTER TABLE sets_new RENAME TO sets;

-- Recreate the index for api_id
CREATE INDEX IF NOT EXISTS idx_sets_api_id ON sets(api_id);
