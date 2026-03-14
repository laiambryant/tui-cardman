CREATE TABLE IF NOT EXISTS magic_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL UNIQUE REFERENCES cards(id) ON DELETE CASCADE,
    mana_cost TEXT,
    cmc REAL,
    colors TEXT,
    color_identity TEXT,
    type_line TEXT,
    types TEXT,
    supertypes TEXT,
    subtypes TEXT,
    text TEXT,
    flavor TEXT,
    power TEXT,
    toughness TEXT,
    loyalty TEXT,
    layout TEXT,
    legalities TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_magic_cards_card_id ON magic_cards(card_id);
