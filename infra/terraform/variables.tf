variable "location" {
  description = "Azure region where resources will be deployed"
  type        = string
  default     = "eastus"
}

variable "resource_group_name" {
  description = "Name of the Azure Resource Group"
  type        = string
  default     = "rg-event-ingestor"
}

variable "cluster_name" {
  description = "Name of the AKS cluster"
  type        = string
  default     = "aks-event-ingestor"
}

variable "acr_name" {
  description = "Name of the Azure Container Registry (must be globally unique)"
  type        = string
  default     = null # If null, a random name will be generated
}

variable "node_count" {
  description = "Number of nodes in the AKS default node pool"
  type        = number
  default     = 2
}

variable "vm_size" {
  description = "VM size for the AKS nodes"
  type        = string
  default     = "Standard_B2s" # Cost-effective for demos/dev
}

variable "kubernetes_namespace" {
  description = "Kubernetes namespace to deploy the application"
  type        = string
  default     = "ingestor"
}

variable "image_repository" {
  description = "The image repository name (e.g. go-event-ingestor)"
  type        = string
  default     = "go-event-ingestor"
}

variable "image_tag" {
  description = "The tag of the image to deploy"
  type        = string
  default     = "latest"
}

variable "app_replicas" {
  description = "Number of replicas for the application deployment"
  type        = number
  default     = 2
}

variable "app_env" {
  description = "Map of environment variables to pass to the application"
  type        = map(string)
  default     = {
    HTTP_PORT         = "8080"
    LOG_LEVEL         = "INFO"
    KAFKA_BROKERS     = "my-kafka-broker:9092"
    KAFKA_TOPIC       = "events"
    KAFKA_DLQ_TOPIC   = "events-dlq"
    WORKER_POOL_SIZE  = "10"
  }
}
