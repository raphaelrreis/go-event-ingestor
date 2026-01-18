module "app" {
  source = "../modules/k8s-app"

  namespace = var.kubernetes_namespace
  app_name  = "go-event-ingestor"
  image     = "${azurerm_container_registry.acr.login_server}/${var.image_repository}:${var.image_tag}"
  replicas  = var.app_replicas
  env_vars  = var.app_env
  port      = 8080

  depends_on = [azurerm_kubernetes_cluster.aks]
}
