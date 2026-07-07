# Always-on Postgres — Cloud SQL has no scale-to-zero tier. Reached from Cloud Run
# via a Cloud SQL Auth Proxy sidecar (see cloud-run-bap.tf / cloud-run-bpp.tf),
# not the native /cloudsql socket mount, so bap/bpp need zero code changes
# (BAP_DB_HOST/BPP_DB_HOST just become 127.0.0.1, a normal TCP connection).

resource "google_sql_database_instance" "postgres" {
  name                = "insurance-postgres"
  database_version    = "POSTGRES_16"
  region              = var.region
  deletion_protection = var.sql_deletion_protection

  settings {
    tier    = var.postgres_tier
    edition = "ENTERPRISE" # db-f1-micro (shared-core) isn't valid under this project's ENTERPRISE_PLUS default

    ip_configuration {
      ipv4_enabled    = false
      private_network = google_compute_network.vpc.id
    }

    backup_configuration {
      enabled = false
    }
  }

  depends_on = [google_service_networking_connection.private_vpc_connection]
}

resource "google_sql_database" "bap" {
  name     = var.db_name_bap
  instance = google_sql_database_instance.postgres.name
}

resource "google_sql_database" "bpp" {
  name     = var.db_name_bpp
  instance = google_sql_database_instance.postgres.name
}

resource "google_sql_user" "insurance" {
  name     = var.db_user
  instance = google_sql_database_instance.postgres.name
  password = random_password.postgres.result
}
