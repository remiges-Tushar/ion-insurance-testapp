# cs-mock -> bpp is one-directional (bpp never references cs-mock), so BPP_WEBHOOK_URL
# is safe to set directly with no wiring-step patch needed.

resource "google_cloud_run_v2_service" "cs" {
  name                = "cs-mock"
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
      name  = "cs-mock"
      image = local.images.cs

      ports {
        container_port = 9090
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }

      env {
        name  = "CS_PORT"
        value = "9090"
      }
      env {
        name  = "BPP_WEBHOOK_URL"
        value = "${google_cloud_run_v2_service.bpp.uri}/webhook"
      }
    }
  }

  lifecycle {
    ignore_changes = [template[0].containers[0].image]
  }
}
