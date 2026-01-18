# Shared Terraform Modules

## Kubernetes Application (`infra/modules/k8s-app`)

This module abstracts the standard Kubernetes deployment logic to ensure consistency across AWS, GCP, and Azure clusters.

### Responsibility
- Creating the `Namespace` (if not exists).
- Creating the `Deployment` with:
    - Replica count.
    - Resource Limits/Requests.
    - Liveness/Readiness Probes.
    - Environment Variables injection.
- Creating the `Service` (LoadBalancer type).

### Inputs (Variables)

| Name | Type | Default | Description |
|---|---|---|---|
| `app_name` | string | `go-event-ingestor` | Name of the deployment/service. |
| `image` | string | **Required** | Full container image URI (registry + tag). |
| `replicas` | number | `2` | Number of pod replicas. |
| `env_vars` | map(string) | `{}` | Map of environment variables for the container. |

### Outputs

| Name | Description |
|---|---|
| `service_ip` | The external IP address of the LoadBalancer (once provisioned). |

### Usage Example

```hcl
module "app" {
  source = "../modules/k8s-app"

  app_name = "my-app"
  image    = "my-registry/my-app:v1.0.0"
  env_vars = {
    DB_HOST = "localhost"
  }
}
```
