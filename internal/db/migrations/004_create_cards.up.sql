CREATE TABLE cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_game_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    rarity TEXT,
    is_placeholder BOOLEAN DEFAULT false,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    api_id TEXT UNIQUE,
    set_id INTEGER,
    number TEXT,
    artist TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_game_id) REFERENCES card_games(id),
    FOREIGN KEY (set_id) REFERENCES sets(id)
);

CREATE INDEX idx_cards_api_id ON cards(api_id);
CREATE INDEX idx_cards_set_id ON cards(set_id);
CREATE UNIQUE INDEX idx_cards_set_number ON cards(set_id, number);