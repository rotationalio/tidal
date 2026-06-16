-- Simple table for testing the fields package.
CREATE TABLE IF NOT EXISTS testing (
    id    INTEGER PRIMARY KEY AUTOINCREMENT,
    alpha BLOB NOT NULL,                        -- Used for JSONB and StringArray fields
    bravo BLOB DEFAULT NULL                     -- Used for NullJSONB and NullStringArray fields
);