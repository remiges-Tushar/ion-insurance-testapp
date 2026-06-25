ALTER TABLE policies
  ADD COLUMN IF NOT EXISTS doku_invoice_number VARCHAR(255),
  ADD COLUMN IF NOT EXISTS doku_request_id     VARCHAR(255),
  ADD COLUMN IF NOT EXISTS doku_va_number      VARCHAR(50),
  ADD COLUMN IF NOT EXISTS doku_qris_string    TEXT,
  ADD COLUMN IF NOT EXISTS payment_received    BOOLEAN NOT NULL DEFAULT false;

---- create above / drop below ----

ALTER TABLE policies
  DROP COLUMN IF EXISTS doku_invoice_number,
  DROP COLUMN IF EXISTS doku_request_id,
  DROP COLUMN IF EXISTS doku_va_number,
  DROP COLUMN IF EXISTS doku_qris_string,
  DROP COLUMN IF EXISTS payment_received;
