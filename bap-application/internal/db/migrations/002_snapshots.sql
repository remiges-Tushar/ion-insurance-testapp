CREATE TABLE contract_snapshots (
  id              BIGSERIAL    PRIMARY KEY,
  transaction_id  VARCHAR(255) NOT NULL REFERENCES transactions(transaction_id),
  on_action       VARCHAR(50)  NOT NULL,
  payload         JSONB        NOT NULL DEFAULT '{}',
  received_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

---- create above / drop below ----

DROP TABLE IF EXISTS contract_snapshots;
