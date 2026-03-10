CREATE TABLE IF NOT EXISTS pokemon_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL UNIQUE REFERENCES cards(id) ON DELETE CASCADE,
    hp INTEGER,
    category TEXT,
    stage TEXT,
    evolve_from TEXT,
    description TEXT,
    level TEXT,
    retreat INTEGER,
    regulation_mark TEXT,
    legal_standard BOOLEAN DEFAULT false,
    legal_expanded BOOLEAN DEFAULT false,
    types TEXT,
    attacks TEXT,
    abilities TEXT,
    weaknesses TEXT,
    resistances TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_pokemon_cards_card_id ON pokemon_cards(card_id);
