ALTER TABLE transactions
  ADD COLUMN IF NOT EXISTS doku_invoice_number VARCHAR(255),
  ADD COLUMN IF NOT EXISTS doku_va_number      VARCHAR(50),
  ADD COLUMN IF NOT EXISTS doku_qris_string    TEXT,
  ADD COLUMN IF NOT EXISTS payment_amount      BIGINT;

---- create above / drop below ----

ALTER TABLE transactions
  DROP COLUMN IF EXISTS doku_invoice_number,
  DROP COLUMN IF EXISTS doku_va_number,
  DROP COLUMN IF EXISTS doku_qris_string,
  DROP COLUMN IF EXISTS payment_amount;
