-- Simple table for testing the fields package.
CREATE TABLE IF NOT EXISTS testing (
    id      SERIAL PRIMARY KEY,
    alpha   BYTEA NOT NULL,         -- Used for JSONB and StringArray fields
    bravo   BYTEA DEFAULT NULL,     -- Used for NullJSONB and NullStringArray fields
    charlie JSONB NOT NULL,         -- Used for JSONB testing with JSONB postgres type
    delta   JSONB DEFAULT NULL      -- Used for NullJSONB testing with JSONB postgres type
);

-- Additional table for testing the fields package.
CREATE TABLE IF NOT EXISTS timeseries (
    id      SERIAL PRIMARY KEY,
    alpha   TIMESTAMP DEFAULT NULL,            -- Used for Timestamp field
    bravo   TIMESTAMP NOT NULL DEFAULT NOW(),  -- Used for Timestamp field
    charlie TIMESTAMPTZ DEFAULT NULL,          -- Used for Timestamp field
    delta   TIMESTAMPTZ NOT NULL DEFAULT NOW() -- Used for Timestamp field
);