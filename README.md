# Go Event Ingestor (Multi-Cloud)

A high-performance, stateless HTTP event ingestion service designed for Multi-Cloud deployments (AWS, GCP, Azure).

## 🚀 Overview

This project demonstrates a production-grade **Infrastructure as Code (IaC)** setup for deploying a Go application to three major cloud providers using **Terraform**. The application logic remains identical; only the infrastructure layer adapts to the specific cloud provider.

**Core Application:**
- HTTP API (POST /events)
- Kafka Producer (Main Topic + DLQ)
- Prometheus Metrics

## 🔄 Professional Workflow (CI/CD)

We enforce a strict **"Local == CI"** policy.
GitHub Actions runs the **exact same command** you should run locally before pushing.

### 1. The Golden Rule
**NEVER push code without running:**
```bash
make ci
```

### 2. What `make ci` does
It runs the full validation pipeline in order:
1.  `go mod tidy` (Dependency cleanup)
2.  `go fmt` (Code formatting)
3.  `golangci-lint` (Strict linting)
4.  `go test` (Unit tests with race detection)
5.  `go test -tags=integration` (Integration tests with Docker)
6.  `go build` (Binary compilation)
7.  `docker build` (Container packaging)

If **ANY** step fails, the pipeline aborts immediately.

### 3. Pre-Commit Hook (Recommended)
To prevent accidental bad commits, set up a git hook:

```bash
echo "#!/bin/sh\nmake ci" > .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

## 🏗 Architecture

### Generic Application Layer (Kubernetes)
The application runs as a stateless Deployment on Kubernetes. We use a **Shared Terraform Module** (`infra/modules/k8s-app`) to define the K8s resources (Deployment, Service, Ingress) once and reuse them across EKS, GKE, and AKS.

### Cloud Specifics

| Provider | Kubernetes | Registry | Network |
|---|---|---|---|
| **AWS** | EKS (Elastic Kubernetes Service) | ECR | VPC |
| **GCP** | GKE (Google Kubernetes Engine) | Artifact Registry | VPC |
| **Azure**| AKS (Azure Kubernetes Service) | ACR | VNet |

## 📂 Project Structure

```
.
├── cmd/                # Go Application Entrypoint
├── infra/
│   ├── modules/
│   │   └── k8s-app/    # Shared K8s Deployment Module (DRY)
│   ├── aws/            # Terraform for AWS (EKS + ECR)
│   ├── gcp/            # Terraform for GCP (GKE + GAR)
│   └── azure/          # Terraform for Azure (AKS + ACR)
├── tests/              # Integration tests
├── Dockerfile          # Multi-stage build
├── docker-compose.yml  # Local dev stack (Kafka + Prometheus)
└── Makefile            # Commands
```

## 🛠 Local Development

1.  **Start dependencies**:
    ```bash
    docker-compose up -d
    ```
2.  **Run App**:
    ```bash
    make run
    ```
3.  **Test**:
    ```bash
    curl -X POST http://localhost:8080/events -d '{"type":"test"}'
    ```

## ☁️ Deployment Guide

### Prerequisites
- Terraform >= 1.5
- Cloud CLI (awscli, gcloud, az)
- Docker

### 1. AWS Deployment
```bash
cd infra/aws
terraform init
terraform apply -var="region=us-east-1"
# Output provides ECR URL and kubectl config command
```

### 2. GCP Deployment
```bash
cd infra/gcp
terraform init
terraform apply -var="project_id=YOUR_PROJECT_ID"
# Output provides GAR URL and gcloud credentials command
```

### 3. Azure Deployment
```bash
cd infra/azure
terraform init
terraform apply
# Output provides ACR URL and az aks credentials command
```

## 🧠 Design Decisions & Trade-offs

1.  **Shared Module vs. Cloud Specifics**:
    - We abstracted the *application* deployment into `infra/modules/k8s-app` because Kubernetes manifests are cloud-agnostic.
    - We kept network and cluster creation specific (`infra/aws`, `infra/azure`) because trying to abstract VPCs vs VNets leads to leaky abstractions and over-engineering.

2.  **External Kafka**:
    - This project does *not* provision Kafka via Terraform to keep costs and complexity manageable. In a real scenario, you would use AWS MSK, Confluent Cloud, or Azure Event Hubs and pass the connection string via `app_env` variables in Terraform.

3.  **State Management**:
    - Local state is used for simplicity. In production, use S3/GCS/Azure Storage backends.

## 📝 License
MIT
