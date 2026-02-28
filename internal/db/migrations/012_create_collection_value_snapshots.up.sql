CREATE TABLE IF NOT EXISTS collection_value_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    card_game_id INTEGER NOT NULL,
    total_value REAL NOT NULL DEFAULT 0,
    snapshot_date DATE NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (card_game_id) REFERENCES card_games(id),
    UNIQUE(user_id, card_game_id, snapshot_date)
);
CREATE INDEX IF NOT EXISTS idx_value_snapshots_user_game ON collection_value_snapshots(user_id, card_game_id);
