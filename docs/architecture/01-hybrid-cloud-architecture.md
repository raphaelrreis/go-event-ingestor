# RFC 01: Hybrid Cloud Architecture

| Status        | Implemented |
| :---          | :--- |
| **Feature**   | Multi-Cloud Infrastructure (AWS, GCP, Azure) |
| **Date**      | 2026-01-15 |

---

## 1. Executive Summary

This architecture enables the **Go Event Ingestor** to be deployed seamlessly across three major cloud providers: **AWS**, **GCP**, and **Azure**.

The goal is to demonstrate **Cloud Agnosticism** at the application layer while using **Idiomatic Infrastructure** (Terraform) for each cloud provider.

---

## 2. Business Context

In a modern enterprise environment, avoiding vendor lock-in is a strategic priority. This project serves as a reference implementation for:
- **Disaster Recovery**: Running active-active across clouds.
- **Cost Arbitrage**: Moving workloads to the cheapest provider.
- **Compliance**: Storing data in specific regions/clouds due to regulations.

---

## 3. Architecture

### 3.1 The "Hub and Spoke" Pattern

We use a modular Terraform approach to maximize code reuse without creating leaky abstractions.

*   **The Hub (Application)**: A shared Terraform module (`infra/modules/k8s-app`) defines the Kubernetes resources (Deployment, Service, HPA). This code is 100% cloud-agnostic.
*   **The Spokes (Infrastructure)**: Specific folders (`infra/aws`, `infra/gcp`, `infra/azure`) handle the unique networking and cluster provisioning for each provider.

### 3.2 Component Mapping

| Component | AWS | GCP | Azure |
|---|---|---|---|
| **Compute** | EKS (Elastic Kubernetes Service) | GKE (Google Kubernetes Engine) | AKS (Azure Kubernetes Service) |
| **Registry** | ECR (Elastic Container Registry) | GAR (Google Artifact Registry) | ACR (Azure Container Registry) |
| **Network** | VPC | VPC | VNet |
| **Identity** | IAM Roles for Service Accounts | Workload Identity | Managed Identities |

---

## 4. Ingestion Pipeline

The application follows a **Hexagonal Architecture**:

1.  **Input Port**: HTTP Server (POST `/events`).
2.  **Domain Logic**: Event validation, rate limiting (Token Bucket).
3.  **Output Port**: Kafka Producer (Async, Batched).

```ascii
[Client] -> (HTTP) -> [Ingestor Service] -> (Kafka Protocol) -> [External Kafka]
                           |
                      [Prometheus]
```

---

## 5. Deployment Strategy

### 5.1 Immutable Infrastructure
- We do not ssh into servers.
- We do not change config manually (ClickOps).
- All changes go through `terraform apply`.

### 5.2 CI/CD
- **Local**: `make ci` runs the full validation pipeline.
- **GitHub Actions**: Enforces the same checks.
- **Artifacts**: Docker images are built and pushed to the cloud-specific registry.

---

## 6. Trade-offs

| Decision | Trade-off |
| :--- | :--- |
| **Managed K8s** | + Less ops overhead. <br> - Higher cost than EC2/VMs. |
| **External Kafka** | + We don't manage stateful clusters. <br> - Latency depends on network peering. |
| **Terraform Modules** | + DRY application code. <br> - Complexity in passing variables between layers. |

---

## 7. Future Work

- **Service Mesh**: Linkerd/Istio for cross-cloud traffic.
- **Global Load Balancer**: Cloudflare or similar to route traffic to the nearest cloud.
- **GitOps**: ArgoCD for managing the K8s manifests instead of direct Terraform apply.
