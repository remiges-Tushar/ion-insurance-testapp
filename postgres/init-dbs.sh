#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
  CREATE DATABASE insurance_bpp;
  CREATE DATABASE insurance_bap;
  GRANT ALL PRIVILEGES ON DATABASE insurance_bpp TO $POSTGRES_USER;
  GRANT ALL PRIVILEGES ON DATABASE insurance_bap TO $POSTGRES_USER;
EOSQL
