# NOTE: bap-frontend <-> bap and bap-frontend <-> ion are mutual references —
# see cloud-run-wiring.tf, which patches BAP_URL and ION_URL once every
# service's URL is known. BPP_URL is safe to set directly (bpp never
# references bap-frontend back).

resource "google_cloud_run_v2_service" "bap_frontend" {
  name                = "insurance-bap"
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
      name  = "insurance-bap"
      image = local.images.bap_frontend

      ports {
        container_port = 80
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }

      env {
        name  = "BPP_URL"
        value = google_cloud_run_v2_service.bpp.uri
      }
      env {
        name  = "NGINX_RESOLVER"
        value = "8.8.8.8 8.8.4.4"
      }
      # Patched post-apply — see cloud-run-wiring.tf.
      env {
        name  = "BAP_URL"
        value = ""
      }
      env {
        name  = "ION_URL"
        value = ""
      }
    }
  }

  lifecycle {
    ignore_changes = [template[0].containers[0].image]
  }
}
