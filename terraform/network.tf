# VPC used for: (1) Direct VPC Egress from Cloud Run to Memorystore, and
# (2) the private services peering Cloud SQL needs for its private IP.

resource "google_compute_network" "vpc" {
  name                    = "insurance-app-vpc"
  auto_create_subnetworks = false
  depends_on              = [google_project_service.apis]
}

resource "google_compute_subnetwork" "subnet" {
  name          = "insurance-app-subnet"
  region        = var.region
  network       = google_compute_network.vpc.id
  ip_cidr_range = "10.10.0.0/24"
}

# Shared private-services peering range, used by both Cloud SQL (private IP)
# and Memorystore (which also peers via servicenetworking.googleapis.com).
resource "google_compute_global_address" "private_service_range" {
  name          = "insurance-app-private-service-range"
  purpose       = "VPC_PEERING"
  address_type  = "INTERNAL"
  prefix_length = 16
  network       = google_compute_network.vpc.id
}

resource "google_service_networking_connection" "private_vpc_connection" {
  network                 = google_compute_network.vpc.id
  service                 = "servicenetworking.googleapis.com"
  reserved_peering_ranges = [google_compute_global_address.private_service_range.name]
}
