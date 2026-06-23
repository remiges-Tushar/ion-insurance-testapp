CREATE TABLE resources (
  id                   SERIAL PRIMARY KEY,
  bpp_id               VARCHAR(255) NOT NULL,
  product_type         VARCHAR(50)  NOT NULL,
  vehicle_type         VARCHAR(50)  NOT NULL,
  ojk_product_code     VARCHAR(100) NOT NULL,
  resource_attributes  JSONB        NOT NULL DEFAULT '{}',
  created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE offers (
  id                 SERIAL PRIMARY KEY,
  resource_id        INTEGER      NOT NULL REFERENCES resources(id),
  bpp_id             VARCHAR(255) NOT NULL,
  tariff_zone        VARCHAR(20)  NOT NULL,
  premium_rate_min   NUMERIC(5,2) NOT NULL,
  premium_rate_max   NUMERIC(5,2) NOT NULL,
  offer_attributes   JSONB        NOT NULL DEFAULT '{}',
  valid_until        TIMESTAMPTZ
);

CREATE TABLE offer_considerations (
  id               SERIAL PRIMARY KEY,
  offer_id         INTEGER      NOT NULL REFERENCES offers(id),
  breakup          JSONB        NOT NULL DEFAULT '[]',
  total_premium_idr BIGINT      NOT NULL DEFAULT 0,
  currency         VARCHAR(10)  NOT NULL DEFAULT 'IDR'
);

---- create above / drop below ----

DROP TABLE IF EXISTS offer_considerations;
DROP TABLE IF EXISTS offers;
DROP TABLE IF EXISTS resources;
