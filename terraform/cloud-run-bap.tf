# NOTE on env vars: bap <-> onix-bap and bap <-> bap-frontend are mutual
# references (bap needs onix-bap's caller URL and bap-frontend's URL; both of
# those in turn need bap's URL), which Terraform cannot express without a
# dependency cycle. BAP_ONIX_BAP_CALLER_URL and BAP_FRONTEND_URL are therefore
# left blank here and patched in by cloud-run-wiring.tf once every service's
# URL is known. ION_SERVICE_URL is safe to set directly — ion never references
# bap back, so there's no cycle there.

resource "google_cloud_run_v2_service" "bap" {
  name                = "bap"
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
      name  = "bap"
      image = local.images.bap

      depends_on = ["cloudsql-proxy"]

      ports {
        container_port = 8083
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }

      env {
        name  = "BAP_SERVER_PORT"
        value = "8083"
      }
      env {
        name  = "BAP_DB_HOST"
        value = "127.0.0.1"
      }
      env {
        name  = "BAP_DB_PORT"
        value = "5432"
      }
      env {
        name  = "BAP_DB_USER"
        value = var.db_user
      }
      env {
        name = "BAP_DB_PASSWORD"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.this["postgres-password"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "BAP_DB_NAME"
        value = var.db_name_bap
      }
      env {
        name  = "BAP_DB_SSLMODE"
        value = "disable"
      }
      env {
        name  = "BAP_REDIS_ADDR"
        value = "${google_redis_instance.cache.host}:${google_redis_instance.cache.port}"
      }
      env {
        name  = "ION_SERVICE_URL"
        value = google_cloud_run_v2_service.ion.uri
      }
      # Patched post-apply — see cloud-run-wiring.tf.
      env {
        name  = "BAP_ONIX_BAP_CALLER_URL"
        value = ""
      }
      env {
        name  = "BAP_FRONTEND_URL"
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
