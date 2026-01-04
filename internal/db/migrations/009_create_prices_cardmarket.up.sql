-- Create prices_cardmarket table for Cardmarket price snapshots
CREATE TABLE IF NOT EXISTS prices_cardmarket (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    avg_price REAL,
    trend_price REAL,
    url TEXT,
    snapshot_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_prices_cardmarket_card_id ON prices_cardmarket(card_id);
CREATE INDEX IF NOT EXISTS idx_prices_cardmarket_snapshot_at ON prices_cardmarket(snapshot_at);
