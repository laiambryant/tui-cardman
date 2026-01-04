-- Create prices_tcgplayer table for TCGPlayer price snapshots
CREATE TABLE IF NOT EXISTS prices_tcgplayer (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    price_type TEXT NOT NULL,
    low REAL,
    mid REAL,
    high REAL,
    market REAL,
    direct_low REAL,
    tcgplayer_url TEXT,
    tcgplayer_updated_at TEXT,
    snapshot_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_prices_tcgplayer_card_id ON prices_tcgplayer(card_id);
CREATE INDEX IF NOT EXISTS idx_prices_tcgplayer_snapshot_at ON prices_tcgplayer(snapshot_at);
