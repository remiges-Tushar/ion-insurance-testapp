ALTER TABLE transactions
  ADD COLUMN IF NOT EXISTS doku_checkout_request_id VARCHAR(255);

---- create above / drop below ----

ALTER TABLE transactions
  DROP COLUMN IF EXISTS doku_checkout_request_id;
