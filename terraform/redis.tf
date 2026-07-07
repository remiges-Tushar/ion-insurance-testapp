# Always-on Redis — Memorystore has no scale-to-zero tier either. Reached from
# Cloud Run via Direct VPC Egress (no separate Serverless VPC Access connector
# resource needed — see the vpc_access blocks on bap/onix-bap/onix-bpp).

resource "google_redis_instance" "cache" {
  name               = "insurance-redis"
  tier               = var.redis_tier
  memory_size_gb     = var.redis_memory_size_gb
  region             = var.region
  authorized_network = google_compute_network.vpc.id
  redis_version      = "REDIS_7_0"

  depends_on = [google_service_networking_connection.private_vpc_connection]
}
