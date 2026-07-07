ALTER TABLE policies
  ADD COLUMN IF NOT EXISTS reconcile_status VARCHAR(50) NOT NULL DEFAULT 'PENDING';

---- create above / drop below ----

ALTER TABLE policies
  DROP COLUMN IF EXISTS reconcile_status;
