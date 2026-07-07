# NOTE: onix-bpp <-> bpp and onix-bpp <-> onix-bap are mutual references —
# see cloud-run-wiring.tf, which patches BPP_URL and ONIX_BAP_URL once every
# service's URL is known. CS_URL is safe to set directly (cs-mock never
# references onix-bpp back).

resource "google_cloud_run_v2_service" "onix_bpp" {
  name                = "onix-bpp"
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
      name  = "onix-bpp"
      image = local.images.onix_bpp

      ports {
        container_port = 8082
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }

      env {
        name  = "REDIS_ADDR"
        value = "${google_redis_instance.cache.host}:${google_redis_instance.cache.port}"
      }
      env {
        name  = "CS_URL"
        value = google_cloud_run_v2_service.cs.uri
      }
      env {
        name = "BPP_SIGNING_PRIVATE_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.this["bpp-signing-private-key"].secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "BPP_SIGNING_PUBLIC_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.this["bpp-signing-public-key"].secret_id
            version = "latest"
          }
        }
      }
      # Patched post-apply — see cloud-run-wiring.tf.
      env {
        name  = "BPP_URL"
        value = ""
      }
      env {
        name  = "ONIX_BAP_URL"
        value = ""
      }
    }
  }

  lifecycle {
    ignore_changes = [template[0].containers[0].image]
  }
}
