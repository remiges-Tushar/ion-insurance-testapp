output "bap_frontend_url" {
  value       = google_cloud_run_v2_service.bap_frontend.uri
  description = "Public URL — open this to use the BAP (buyer) app."
}

output "bpp_frontend_url" {
  value       = google_cloud_run_v2_service.bpp_frontend.uri
  description = "Public URL — open this to use the BPP (seller) app."
}

output "bap_url" {
  value = google_cloud_run_v2_service.bap.uri
}

output "bpp_url" {
  value = google_cloud_run_v2_service.bpp.uri
}

output "onix_bap_url" {
  value = google_cloud_run_v2_service.onix_bap.uri
}

output "onix_bpp_url" {
  value = google_cloud_run_v2_service.onix_bpp.uri
}

output "ion_url" {
  value = google_cloud_run_v2_service.ion.uri
}

output "cs_mock_url" {
  value = google_cloud_run_v2_service.cs.uri
}

output "artifact_registry_repo" {
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${var.artifact_registry_repo_id}"
  description = "Push images here — see scripts/build-and-push.sh."
}

output "cloud_sql_connection_name" {
  value = google_sql_database_instance.postgres.connection_name
}

output "redis_host" {
  value = google_redis_instance.cache.host
}
