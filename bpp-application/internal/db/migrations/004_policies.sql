CREATE TYPE policy_status AS ENUM (
  'PENDING_ISSUANCE',
  'ACTIVE',
  'CANCELLED',
  'EXPIRED',
  'TOTAL_LOSS'
);

CREATE TABLE policies (
  id               SERIAL PRIMARY KEY,
  transaction_id   VARCHAR(255) NOT NULL UNIQUE,
  bap_id           VARCHAR(255) NOT NULL,
  bpp_id           VARCHAR(255) NOT NULL,
  status           policy_status NOT NULL DEFAULT 'PENDING_ISSUANCE',
  policyholder_nik VARCHAR(16),
  vehicle_vin      VARCHAR(50),
  idv              BIGINT,
  policy_number    VARCHAR(100),
  certificate_url  VARCHAR(500),
  coverage_start   TIMESTAMPTZ,
  coverage_end     TIMESTAMPTZ,
  created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE payments (
  id           SERIAL PRIMARY KEY,
  policy_id    INTEGER      NOT NULL REFERENCES policies(id),
  method       VARCHAR(50),
  payment_ref  VARCHAR(255),
  amount_idr   BIGINT,
  paid_at      TIMESTAMPTZ
);

CREATE TABLE claims (
  id                   SERIAL PRIMARY KEY,
  policy_id            INTEGER      NOT NULL REFERENCES policies(id),
  claim_id             VARCHAR(100) NOT NULL,
  incident_type        VARCHAR(100),
  status               VARCHAR(50)  NOT NULL DEFAULT 'FILED',
  estimated_damage_idr BIGINT,
  filed_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

---- create above / drop below ----

DROP TABLE IF EXISTS claims;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS policies;
DROP TYPE  IF EXISTS policy_status;
