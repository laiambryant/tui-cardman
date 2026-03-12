CREATE TABLE IF NOT EXISTS yugioh_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL UNIQUE REFERENCES cards(id) ON DELETE CASCADE,
    card_type TEXT,
    frame_type TEXT,
    description TEXT,
    atk INTEGER,
    def INTEGER,
    level INTEGER,
    attribute TEXT,
    race TEXT,
    scale INTEGER,
    link_val INTEGER,
    link_markers TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_yugioh_cards_card_id ON yugioh_cards(card_id);
