ALTER TABLE policies
  ADD COLUMN IF NOT EXISTS xendit_va_id VARCHAR(255),
  ADD COLUMN IF NOT EXISTS bank_code    VARCHAR(20);

---- create above / drop below ----

ALTER TABLE policies
  DROP COLUMN IF EXISTS xendit_va_id,
  DROP COLUMN IF EXISTS bank_code;
