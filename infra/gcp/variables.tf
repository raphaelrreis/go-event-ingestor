variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "region" {
  description = "GCP Region"
  type        = string
  default     = "us-central1"
}

variable "cluster_name" {
  description = "GKE Cluster Name"
  type        = string
  default     = "gke-event-ingestor"
}

variable "image_repository" {
  description = "Artifact Registry Repository Name"
  type        = string
  default     = "go-event-ingestor"
}

variable "image_tag" {
  description = "Image tag"
  type        = string
  default     = "latest"
}

variable "app_replicas" {
  description = "Number of replicas"
  type        = number
  default     = 2
}

variable "app_env" {
  description = "Environment variables"
  type        = map(string)
  default     = {
    HTTP_PORT = "8080"
  }
}
