output "resource_group_name" {
  value = azurerm_resource_group.rg.name
}

output "aks_cluster_name" {
  value = azurerm_kubernetes_cluster.aks.name
}

output "acr_login_server" {
  value = azurerm_container_registry.acr.login_server
}

output "get_credentials_command" {
  value = "az aks get-credentials --resource-group ${azurerm_resource_group.rg.name} --name ${azurerm_kubernetes_cluster.aks.name}"
}

output "application_public_ip" {
  value = try(kubernetes_service.ingestor.status.0.load_balancer.0.ingress.0.ip, "Pending (Run 'kubectl get svc -n ${var.kubernetes_namespace}' later)")
}
