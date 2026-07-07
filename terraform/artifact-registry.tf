resource "google_artifact_registry_repository" "repo" {
  location      = var.region
  repository_id = var.artifact_registry_repo_id
  format        = "DOCKER"
  depends_on    = [google_project_service.apis]
}
