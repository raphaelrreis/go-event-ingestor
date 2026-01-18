variable "region" {
  description = "AWS Region"
  type        = string
  default     = "us-east-1"
}

variable "cluster_name" {
  description = "EKS Cluster Name"
  type        = string
  default     = "eks-event-ingestor"
}

variable "app_replicas" {
  description = "Number of replicas"
  type        = number
  default     = 2
}

variable "image_repository" {
  description = "ECR Repository Name"
  type        = string
  default     = "go-event-ingestor"
}

variable "image_tag" {
  description = "Image tag"
  type        = string
  default     = "latest"
}

variable "app_env" {
  description = "Environment variables"
  type        = map(string)
  default     = {
    HTTP_PORT = "8080"
  }
}
