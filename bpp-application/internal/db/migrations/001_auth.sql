CREATE TABLE provider_accounts (
  id            SERIAL PRIMARY KEY,
  company_name  VARCHAR(255) NOT NULL,
  ojk_license   VARCHAR(100) NOT NULL UNIQUE,
  email         VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

---- create above / drop below ----

DROP TABLE IF EXISTS provider_accounts;
