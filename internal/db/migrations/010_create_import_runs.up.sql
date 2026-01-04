-- Create import_runs table to track import operations
CREATE TABLE IF NOT EXISTS import_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    import_type TEXT NOT NULL,
    status TEXT NOT NULL,
    sets_processed INTEGER DEFAULT 0,
    cards_imported INTEGER DEFAULT 0,
    errors_count INTEGER DEFAULT 0,
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    notes TEXT
);

CREATE INDEX IF NOT EXISTS idx_import_runs_status ON import_runs(status);
CREATE INDEX IF NOT EXISTS idx_import_runs_started_at ON import_runs(started_at);
