-- Extend providers: full descriptor + OJK attributes
ALTER TABLE providers ADD COLUMN IF NOT EXISTS descriptor         JSONB NOT NULL DEFAULT '{}';
ALTER TABLE providers ADD COLUMN IF NOT EXISTS provider_attributes JSONB NOT NULL DEFAULT '{}';

-- Link catalogs to a provider row
ALTER TABLE catalogs ADD COLUMN IF NOT EXISTS provider_id INTEGER REFERENCES providers(id);

-- Timestamp on offers for display
ALTER TABLE offers ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

---- create above / drop below ----

ALTER TABLE offers    DROP COLUMN IF EXISTS created_at;
ALTER TABLE catalogs  DROP COLUMN IF EXISTS provider_id;
ALTER TABLE providers DROP COLUMN IF EXISTS provider_attributes;
ALTER TABLE providers DROP COLUMN IF EXISTS descriptor;
