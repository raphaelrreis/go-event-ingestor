variable "namespace" {
  description = "Kubernetes namespace"
  type        = string
  default     = "ingestor"
}

variable "app_name" {
  description = "Application name"
  type        = string
  default     = "go-event-ingestor"
}

variable "image" {
  description = "Full Docker image URI (registry/repo:tag)"
  type        = string
}

variable "replicas" {
  description = "Number of replicas"
  type        = number
  default     = 2
}

variable "port" {
  description = "Container port"
  type        = number
  default     = 8080
}

variable "env_vars" {
  description = "Map of environment variables"
  type        = map(string)
  default     = {}
}
