CREATE TABLE IF NOT EXISTS cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_game_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    expansion TEXT,
    rarity TEXT,
    card_number TEXT,
    release_date DATE,
    is_placeholder BOOLEAN DEFAULT false,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    api_id TEXT UNIQUE,
    set_id INTEGER,
    number TEXT,
    artist TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_game_id) REFERENCES card_games(id)
);

CREATE INDEX IF NOT EXISTS idx_cards_api_id ON cards(api_id);
CREATE INDEX IF NOT EXISTS idx_cards_set_id ON cards(set_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cards_set_number ON cards(set_id, number);

-- Insert some placeholder data for testing
-- Pokemon cards
INSERT INTO cards (card_game_id, name, expansion, rarity, card_number, release_date, is_placeholder) VALUES
  (1, 'Pikachu', 'Base Set', 'Common', '25/102', '1998-10-20', true),
  (1, 'Charizard', 'Base Set', 'Rare Holo', '4/102', '1998-10-20', true),
  (1, 'Blastoise', 'Base Set', 'Rare Holo', '2/102', '1998-10-20', true),
  (1, 'Venusaur', 'Base Set', 'Rare Holo', '15/102', '1998-10-20', true),
  (1, 'Mewtwo', 'Base Set', 'Rare Holo', '10/102', '1998-10-20', true);

-- Magic: The Gathering cards
INSERT INTO cards (card_game_id, name, expansion, rarity, card_number, release_date, is_placeholder) VALUES
  (2, 'Black Lotus', 'Alpha', 'Rare', 'A-232', '1993-08-05', true),
  (2, 'Lightning Bolt', 'Alpha', 'Common', 'A-161', '1993-08-05', true),
  (2, 'Ancestral Recall', 'Alpha', 'Rare', 'A-48', '1993-08-05', true),
  (2, 'Time Walk', 'Alpha', 'Rare', 'A-84', '1993-08-05', true),
  (2, 'Mox Sapphire', 'Alpha', 'Rare', 'A-266', '1993-08-05', true);

-- Yu-Gi-Oh! cards
INSERT INTO cards (card_game_id, name, expansion, rarity, card_number, release_date, is_placeholder) VALUES
  (3, 'Blue-Eyes White Dragon', 'Legend of Blue Eyes White Dragon', 'Ultra Rare', 'LOB-001', '2002-03-08', true),
  (3, 'Dark Magician', 'Legend of Blue Eyes White Dragon', 'Ultra Rare', 'LOB-005', '2002-03-08', true),
  (3, 'Exodia the Forbidden One', 'Legend of Blue Eyes White Dragon', 'Ultra Rare', 'LOB-124', '2002-03-08', true),
  (3, 'Red-Eyes Black Dragon', 'Legend of Blue Eyes White Dragon', 'Ultra Rare', 'LOB-070', '2002-03-08', true),
  (3, 'Summoned Skull', 'Legend of Blue Eyes White Dragon', 'Ultra Rare', 'LOB-113', '2002-03-08', true);