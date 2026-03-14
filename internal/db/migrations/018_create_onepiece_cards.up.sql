CREATE TABLE IF NOT EXISTS onepiece_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL UNIQUE REFERENCES cards(id) ON DELETE CASCADE,
    card_color TEXT,
    card_type TEXT,
    card_text TEXT,
    sub_types TEXT,
    attribute TEXT,
    life TEXT,
    card_cost TEXT,
    card_power TEXT,
    counter_amount TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_onepiece_cards_card_id ON onepiece_cards(card_id);
