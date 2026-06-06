-- Simple table for testing the fields package.
CREATE TABLE IF NOT EXISTS testing (
    id      SERIAL PRIMARY KEY,
    alpha   BYTEA NOT NULL,       -- Used for JSONB and StringArray fields
    bravo   BYTEA DEFAULT NULL,   -- Used for NullJSONB and NullStringArray fields
    charlie JSONB NOT NULL,       -- Used for JSONB testing with JSONB postgres type
    delta   JSONB DEFAULT NULL    -- Used for NullJSONB testing with JSONB postgres type
);