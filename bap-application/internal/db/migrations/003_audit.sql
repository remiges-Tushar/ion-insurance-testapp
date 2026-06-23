CREATE TABLE beckn_message_log (
  id             BIGSERIAL,
  action         VARCHAR(50)  NOT NULL,
  transaction_id VARCHAR(255),
  payload        JSONB        NOT NULL DEFAULT '{}',
  created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

CREATE TABLE beckn_message_log_2026_01 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE beckn_message_log_2026_02 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE beckn_message_log_2026_03 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE beckn_message_log_2026_04 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE beckn_message_log_2026_05 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE beckn_message_log_2026_06 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE beckn_message_log_2026_07 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE beckn_message_log_2026_08 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE beckn_message_log_2026_09 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE beckn_message_log_2026_10 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE beckn_message_log_2026_11 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE beckn_message_log_2026_12 PARTITION OF beckn_message_log
  FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

---- create above / drop below ----

DROP TABLE IF EXISTS beckn_message_log;
