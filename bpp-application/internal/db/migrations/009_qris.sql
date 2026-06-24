ALTER TABLE policies
  ADD COLUMN IF NOT EXISTS xendit_qris_id     VARCHAR(255),
  ADD COLUMN IF NOT EXISTS xendit_qris_string TEXT;

---- create above / drop below ----

ALTER TABLE policies
  DROP COLUMN IF EXISTS xendit_qris_id,
  DROP COLUMN IF EXISTS xendit_qris_string;
