CREATE TABLE IF NOT EXISTS decks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    card_game_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    format TEXT NOT NULL DEFAULT 'standard',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (card_game_id) REFERENCES card_games(id),
    UNIQUE(user_id, card_game_id, name)
);
