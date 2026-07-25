-- cardman:foreign_keys_off

CREATE TABLE sets_v2 (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_game_id INTEGER NOT NULL,
    api_id TEXT NOT NULL,
    code TEXT,
    name TEXT NOT NULL,
    printed_total INTEGER,
    total INTEGER,
    symbol_url TEXT,
    logo_url TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_game_id) REFERENCES card_games(id),
    UNIQUE (card_game_id, api_id)
);

-- Preserve each existing set ID for the first game that references it. Orphaned
-- historical sets pre-date multi-game imports and therefore belong to Pokemon.
INSERT INTO sets_v2 (
    id, card_game_id, api_id, code, name, printed_total, total,
    symbol_url, logo_url, updated_at
)
SELECT
    s.id,
    COALESCE(
        (SELECT MIN(c.card_game_id) FROM cards c WHERE c.set_id = s.id),
        (SELECT id FROM card_games WHERE name = 'Pokemon')
    ),
    s.api_id,
    s.code,
    s.name,
    s.printed_total,
    s.total,
    s.symbol_url,
    s.logo_url,
    s.updated_at
FROM sets s;

-- If a formerly global set row is referenced by more than one game, create a
-- separate row for every additional game before repointing its cards.
INSERT INTO sets_v2 (
    card_game_id, api_id, code, name, printed_total, total,
    symbol_url, logo_url, updated_at
)
SELECT DISTINCT
    c.card_game_id,
    s.api_id,
    s.code,
    s.name,
    s.printed_total,
    s.total,
    s.symbol_url,
    s.logo_url,
    s.updated_at
FROM sets s
JOIN cards c ON c.set_id = s.id
WHERE NOT EXISTS (
    SELECT 1
    FROM sets_v2 migrated
    WHERE migrated.card_game_id = c.card_game_id
      AND migrated.api_id = s.api_id
);

UPDATE cards
SET set_id = (
    SELECT migrated.id
    FROM sets original
    JOIN sets_v2 migrated
      ON migrated.api_id = original.api_id
     AND migrated.card_game_id = cards.card_game_id
    WHERE original.id = cards.set_id
)
WHERE set_id IS NOT NULL;

DROP TABLE sets;
ALTER TABLE sets_v2 RENAME TO sets;

CREATE INDEX idx_sets_api_id ON sets(api_id);
CREATE INDEX idx_sets_card_game_id ON sets(card_game_id);
