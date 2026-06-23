CREATE TABLE transactions (
  transaction_id  VARCHAR(255) PRIMARY KEY,
  action          VARCHAR(50)  NOT NULL,
  status          VARCHAR(50)  NOT NULL DEFAULT 'INITIATED',
  bpp_id          VARCHAR(255),
  created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- status values: SELECT_SENT | QUOTE_RECEIVED | INIT_SENT | INIT_RECEIVED | CONFIRM_SENT | CONFIRMED | CANCELLED

---- create above / drop below ----

DROP TABLE IF EXISTS transactions;
