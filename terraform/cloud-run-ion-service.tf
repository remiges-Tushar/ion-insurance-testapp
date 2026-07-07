# NOTE: ion <-> bpp and ion <-> bap-frontend are mutual references — see
# cloud-run-wiring.tf, which patches BPP_PAYMENT_URL, BAP_FRONTEND_URL, and
# DOKU_CALLBACK_URL once every service's URL is known. DOKU_CALLBACK_URL
# previously pointed at a hardcoded lhr.life dev tunnel — it now points at
# this deployment's own bap-frontend Cloud Run URL, which proxies
# /ion-webhook/ to this service (see bap-frontend/nginx.conf).

resource "google_cloud_run_v2_service" "ion" {
  name                = "ion"
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

    containers {
      name  = "ion"
      image = local.images.ion

      ports {
        container_port = 8090
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }

      env {
        name  = "ION_PORT"
        value = "8090"
      }
      env {
        name = "DOKU_CLIENT_ID"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.this["doku-client-id"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "DOKU_SECRET_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.this["doku-secret-key"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "DOKU_BASE_URL"
        value = "https://api-sandbox.doku.com"
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
        value = var.doku_ion_bank_account_id_on_ion_service
      }
      # Patched post-apply — see cloud-run-wiring.tf.
      env {
        name  = "BPP_PAYMENT_URL"
        value = ""
      }
      env {
        name  = "BAP_FRONTEND_URL"
        value = ""
      }
      env {
        name  = "DOKU_CALLBACK_URL"
        value = ""
      }
    }
  }

  lifecycle {
    ignore_changes = [template[0].containers[0].image]
  }
}
