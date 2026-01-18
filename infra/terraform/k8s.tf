resource "kubernetes_namespace" "app" {
  metadata {
    name = var.kubernetes_namespace
  }
  
  # Ensure AKS is ready before trying to create namespaces
  depends_on = [azurerm_kubernetes_cluster.aks]
}

resource "kubernetes_deployment" "ingestor" {
  metadata {
    name      = "go-event-ingestor"
    namespace = kubernetes_namespace.app.metadata.0.name
    labels = {
      app = "event-ingestor"
    }
  }

  spec {
    replicas = var.app_replicas

    selector {
      match_labels = {
        app = "event-ingestor"
      }
    }

    template {
      metadata {
        labels = {
          app = "event-ingestor"
        }
      }

      spec {
        container {
          image = "${azurerm_container_registry.acr.login_server}/${var.image_repository}:${var.image_tag}"
          name  = "ingestor"

          port {
            container_port = 8080
          }

          # Inject environment variables from the Terraform map variable
          dynamic "env" {
            for_each = var.app_env
            content {
              name  = env.key
              value = env.value
            }
          }

          resources {
            limits = {
              cpu    = "500m"
              memory = "256Mi"
            }
            requests = {
              cpu    = "100m"
              memory = "128Mi"
            }
          }

          liveness_probe {
            http_get {
              path = "/health"
              port = 8080
            }
            initial_delay_seconds = 5
            period_seconds        = 10
          }

          readiness_probe {
            http_get {
              path = "/health"
              port = 8080
            }
            initial_delay_seconds = 5
            period_seconds        = 10
          }
        }
      }
    }
  }
}

resource "kubernetes_service" "ingestor" {
  metadata {
    name      = "go-event-ingestor"
    namespace = kubernetes_namespace.app.metadata.0.name
  }

  spec {
    selector = {
      app = "event-ingestor"
    }

    port {
      port        = 80
      target_port = 8080
    }

    type = "LoadBalancer"
  }
}
