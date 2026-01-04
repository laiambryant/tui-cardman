DROP INDEX IF EXISTS idx_cards_set_number;
DROP INDEX IF EXISTS idx_cards_set_id;
DROP INDEX IF EXISTS idx_cards_api_id;

-- Note: SQLite doesn't support DROP COLUMN directly
-- This would require recreating the table without these columns
-- For now, leaving columns in place for down migration
