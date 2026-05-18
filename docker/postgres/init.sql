CREATE TABLE IF NOT EXISTS schemas (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    version     INTEGER NOT NULL DEFAULT 1,
    fields      JSONB NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(name, version)
);

CREATE INDEX IF NOT EXISTS idx_schemas_name ON schemas(name);
CREATE INDEX IF NOT EXISTS idx_schemas_name_version ON schemas(name, version);