CREATE TABLE catalogs (
  id          SERIAL PRIMARY KEY,
  bpp_id      VARCHAR(255) NOT NULL,
  name        VARCHAR(255) NOT NULL,
  descriptor  JSONB        NOT NULL DEFAULT '{}',
  validity    JSONB        NOT NULL DEFAULT '{}',
  version     VARCHAR(50)  NOT NULL DEFAULT '1.0.0',
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE catalog_publish_results (
  id            SERIAL PRIMARY KEY,
  catalog_id    INTEGER      NOT NULL REFERENCES catalogs(id),
  cds_status    VARCHAR(50)  NOT NULL DEFAULT 'PENDING',
  result_payload JSONB       NOT NULL DEFAULT '{}',
  published_at  TIMESTAMPTZ
);

CREATE TABLE providers (
  id         SERIAL PRIMARY KEY,
  bpp_id     VARCHAR(255) NOT NULL,
  name       VARCHAR(255) NOT NULL,
  locations  JSONB        NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

---- create above / drop below ----

DROP TABLE IF EXISTS providers;
DROP TABLE IF EXISTS catalog_publish_results;
DROP TABLE IF EXISTS catalogs;
