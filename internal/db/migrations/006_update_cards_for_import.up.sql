-- Update cards table to support Pokemon TCG API import
-- Add new columns needed for import functionality

ALTER TABLE cards ADD COLUMN api_id TEXT UNIQUE;
ALTER TABLE cards ADD COLUMN set_id INTEGER REFERENCES sets(id);
ALTER TABLE cards ADD COLUMN number TEXT;
ALTER TABLE cards ADD COLUMN rarity TEXT;
ALTER TABLE cards ADD COLUMN artist TEXT;
ALTER TABLE cards ADD COLUMN flavor_text TEXT; -- keep for now if needed elsewhere
ALTER TABLE cards ADD COLUMN updated_at DATETIME DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_cards_api_id ON cards(api_id);
CREATE INDEX IF NOT EXISTS idx_cards_set_id ON cards(set_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cards_set_number ON cards(set_id, number);
