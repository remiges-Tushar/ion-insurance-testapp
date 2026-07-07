# Every secret Cloud Run reads lives in Secret Manager, never as a plaintext
# env var. Terraform creates the containers for ALL of them (so IAM access
# and references are declarative), but only ever writes a VALUE into the ones
# it can generate itself (just the Postgres password, below). The rest are
# empty containers until you seed them once with scripts/seed-secrets.sh —
# see terraform/README.md. This keeps plaintext secrets out of every
# Terraform-visible file, including .tfvars and state, for anything Terraform
# didn't generate itself.
#
# IMPORTANT ORDERING: a Cloud Run revision fails to deploy if it references a
# secret version that doesn't exist yet. Run scripts/seed-secrets.sh after
# `terraform apply -target=google_secret_manager_secret.this` and BEFORE the
# full `terraform apply` that creates the Cloud Run services.

locals {
  externally_seeded_secret_ids = [
    "bpp-jwt-secret",
    "doku-client-id",
    "doku-secret-key",
    "bap-signing-private-key",
    "bap-signing-public-key",
    "bpp-signing-private-key",
    "bpp-signing-public-key",
  ]

  all_secret_ids = concat(["postgres-password"], local.externally_seeded_secret_ids)
}

resource "google_secret_manager_secret" "this" {
  for_each   = toset(local.all_secret_ids)
  secret_id  = each.key
  depends_on = [google_project_service.apis]

  replication {
    auto {}
  }
}

# The only secret value Terraform ever generates or sees itself.
resource "random_password" "postgres" {
  length  = 32
  special = false
}

resource "google_secret_manager_secret_version" "postgres_password" {
  secret      = google_secret_manager_secret.this["postgres-password"].id
  secret_data = random_password.postgres.result
}

resource "google_secret_manager_secret_iam_member" "accessor" {
  for_each  = google_secret_manager_secret.this
  secret_id = each.value.id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.cloud_run_sa.email}"
}
