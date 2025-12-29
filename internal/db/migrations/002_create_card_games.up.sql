CREATE TABLE IF NOT EXISTS card_games (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO card_games (name) VALUES
  ('Pokemon'),
  ('Magic: The Gathering'),
  ('Yu-Gi-Oh!');
