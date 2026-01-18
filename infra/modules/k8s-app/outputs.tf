output "service_ip" {
  description = "Public IP of the LoadBalancer"
  value       = try(kubernetes_service.app.status.0.load_balancer.0.ingress.0.ip, "Pending")
}

output "namespace" {
  value = kubernetes_namespace.app.metadata.0.name
}
