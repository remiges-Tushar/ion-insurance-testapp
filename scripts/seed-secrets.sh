#!/usr/bin/env bash
# One-time: populate the Secret Manager secrets that terraform/secrets.tf
# creates as empty containers but does not write values into (the Postgres
# password is the only one Terraform generates and writes itself).
#
# Run this AFTER:
#   terraform apply -target=google_secret_manager_secret.this -var-file=terraform.tfvars
# and BEFORE the full `terraform apply` — Cloud Run revisions fail to deploy
# if the secret they reference has no version yet.
#
# Usage: PROJECT_ID=remiges-ion ./scripts/seed-secrets.sh
#
# Defaults below match today's docker-compose.yml / config/insurance-*.yaml
# sandbox values, so a first deploy behaves identically to local
# docker-compose. Override via env vars for real credentials.

set -euo pipefail

PROJECT_ID="${PROJECT_ID:?set PROJECT_ID}"

seed() {
  local name="$1" value="$2"
  printf '%s' "$value" | gcloud secrets versions add "$name" --project="$PROJECT_ID" --data-file=-
}

seed bpp-jwt-secret          "${BPP_JWT_SECRET:-insurance-bpp-jwt-secret-change-in-production}"
seed doku-client-id          "${DOKU_CLIENT_ID:-BRN-0299-1782374867869}"
seed doku-secret-key         "${DOKU_SECRET_KEY:-SK-RZNjjFOg8s8i3FHIoB4m}"
seed bap-signing-private-key "${BAP_SIGNING_PRIVATE_KEY:-ngIGAx7cbZjC9n7+ILCxBuJe8kjx7wHtWAfS+4R5gp8=}"
seed bap-signing-public-key  "${BAP_SIGNING_PUBLIC_KEY:-CnqxJ+Pst4QY92CFEIClhd6cmVXGxMnnVYwL/FvkWFQ=}"
seed bpp-signing-private-key "${BPP_SIGNING_PRIVATE_KEY:-kYgQu14UB1w840W95xmIQF7j9aCuTF8tGj/7dCXrrkY=}"
seed bpp-signing-public-key  "${BPP_SIGNING_PUBLIC_KEY:-SKcAR3CuwaxjVhJWPL3Xe5Pn04qZLD9692lOy3P62qM=}"

echo "done — seeded 7 secrets in ${PROJECT_ID}"
