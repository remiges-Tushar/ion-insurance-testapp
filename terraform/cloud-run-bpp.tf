# NOTE: bpp <-> onix-bpp and bpp <-> ion are mutual references — see
# cloud-run-wiring.tf, which patches BPP_ONIX_BPP_CALLER_URL and
# ION_SERVICE_URL once every service's URL is known. bpp doesn't talk to
# Redis directly (only onix-bpp does), but it still needs vpc_access: its
# cloudsql-proxy sidecar has to reach Cloud SQL's private IP, which is only
# routable from inside the VPC.

resource "google_cloud_run_v2_service" "bpp" {
  name                = "bpp"
  location            = var.region
  ingress             = "INGRESS_TRAFFIC_ALL"
  deletion_protection = false
  depends_on          = [google_project_service.apis]

  template {
    service_account = google_service_account.cloud_run_sa.email

    scaling {
      min_instance_count = 0
      max_instance_count = 3
    }

    vpc_access {
      network_interfaces {
        network    = google_compute_network.vpc.id
        subnetwork = google_compute_subnetwork.subnet.id
      }
      egress = "PRIVATE_RANGES_ONLY"
    }

    containers {
      name  = "bpp"
      image = local.images.bpp

      depends_on = ["cloudsql-proxy"]

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }

      env {
        name  = "BPP_SERVER_PORT"
        value = "8080"
      }
      env {
        name  = "BPP_DB_HOST"
        value = "127.0.0.1"
      }
      env {
        name  = "BPP_DB_PORT"
        value = "5432"
      }
      env {
        name  = "BPP_DB_USER"
        value = var.db_user
      }
      env {
        name = "BPP_DB_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.this["postgres-password"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "BPP_DB_NAME"
        value = var.db_name_bpp
      }
      env {
        name  = "BPP_DB_SSLMODE"
        value = "disable"
      }
      env {
        name = "BPP_JWT_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.this["bpp-jwt-secret"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "DOKU_BPP_BANK_ACCOUNT_ID"
        value = var.doku_bpp_bank_account_id
      }
      env {
        name  = "DOKU_BAP_BANK_ACCOUNT_ID"
        value = var.doku_bap_bank_account_id
      }
      env {
        name  = "DOKU_ION_BANK_ACCOUNT_ID"
        value = var.doku_ion_bank_account_id_on_bpp
      }
      # Patched post-apply — see cloud-run-wiring.tf.
      env {
        name  = "BPP_ONIX_BPP_CALLER_URL"
        value = ""
      }
      env {
        name  = "ION_SERVICE_URL"
        value = ""
      }
    }

    containers {
      name  = "cloudsql-proxy"
      image = local.cloudsql_proxy_image
      args  = ["--structured-logs", "--address=0.0.0.0", "--port=5432", "--private-ip", google_sql_database_instance.postgres.connection_name]

      startup_probe {
        tcp_socket {
          port = 5432
        }
        period_seconds    = 5
        timeout_seconds   = 3
        failure_threshold = 20
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [template[0].containers[0].image]
  }
}
