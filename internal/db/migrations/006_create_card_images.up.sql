-- Create card_images table to store Pokemon TCG card image URLs
CREATE TABLE IF NOT EXISTS card_images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id INTEGER NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    small_url TEXT NOT NULL,
    large_url TEXT NOT NULL,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_card_images_card_id ON card_images(card_id);
