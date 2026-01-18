output "cluster_endpoint" {
  value = google_container_cluster.primary.endpoint
}

output "gar_repository_url" {
  value = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.repo.repository_id}"
}

output "get_credentials_command" {
  value = "gcloud container clusters get-credentials ${var.cluster_name} --region ${var.region} --project ${var.project_id}"
}

output "app_service_ip" {
  value = module.app.service_ip
}
