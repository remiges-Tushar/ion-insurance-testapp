CREATE TABLE support_tickets (
  id          SERIAL PRIMARY KEY,
  policy_id   INTEGER     REFERENCES policies(id),
  description TEXT        NOT NULL,
  status      VARCHAR(50) NOT NULL DEFAULT 'OPEN',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ratings (
  id         SERIAL PRIMARY KEY,
  policy_id  INTEGER     REFERENCES policies(id),
  score      SMALLINT    NOT NULL CHECK (score BETWEEN 1 AND 5),
  feedback   TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

---- create above / drop below ----

DROP TABLE IF EXISTS ratings;
DROP TABLE IF EXISTS support_tickets;
