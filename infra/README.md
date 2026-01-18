# Infrastructure as Code (IaC) - Multi-Cloud Strategy

This directory contains the Terraform configurations to deploy the **Go Event Ingestor** to three major cloud providers: **AWS**, **GCP**, and **Azure**.

## 🎯 Objective

The goal of this infrastructure setup is to demonstrate **Cloud Agnosticism**.
The application logic remains identical; only the infrastructure layer adapts to the specific cloud provider's networking and managed Kubernetes offerings.

---

## 📂 Structure

```
infra/
├── modules/
│   └── k8s-app/    # THE HUB: Shared logic. Defines the Kubernetes Deployment/Service once.
├── aws/            # THE SPOKE: AWS specific networking (VPC) + EKS Cluster.
├── gcp/            # THE SPOKE: GCP specific networking (VPC) + GKE Cluster.
└── azure/          # THE SPOKE: Azure specific networking (VNet) + AKS Cluster.
```

---

## 🚀 How to Deploy

### 1. AWS (Elastic Kubernetes Service)
Best for teams deeply integrated into the AWS ecosystem. Uses `terraform-aws-modules` for best practices.

*   **Resources**: VPC, Private Subnets, EKS Cluster, ECR Repository.
*   **Command**:
    ```bash
    cd infra/aws
    terraform init
    terraform apply -var="region=us-east-1"
    ```

### 2. GCP (Google Kubernetes Engine)
Best for "pure" Kubernetes experiences and speed. Uses GKE Standard (can be switched to Autopilot).

*   **Resources**: VPC, Subnet, GKE Cluster, Artifact Registry.
*   **Command**:
    ```bash
    cd infra/gcp
    terraform init
    terraform apply -var="project_id=YOUR-PROJECT-ID"
    ```

### 3. Azure (Azure Kubernetes Service)
Best for enterprise environments using Active Directory. Uses Managed Identities for security.

*   **Resources**: Resource Group, ACR, AKS Cluster (System Assigned Identity).
*   **Command**:
    ```bash
    cd infra/azure
    terraform init
    terraform apply
    ```

---

## 🧠 Why this Architecture?

### The "Hub and Spoke" Module Pattern
We abstracted the *Application Deployment* into a shared module (`infra/modules/k8s-app`).
*   **Why?** Kubernetes manifests (Deployment, Service, HPA) are standard. Writing them 3 times (once for each cloud) is repetitive and error-prone.
*   **Result**: If we want to change the CPU limit or add an environment variable, we change it **once** in the module, and all 3 clouds are updated.

### Cloud Specifics kept Specific
We did **not** try to abstract the Networking (VPC/VNet) or Cluster creation.
*   **Why?** An AWS VPC is fundamentally different from an Azure VNet. Trying to create a "Generic Network Module" leads to "Leaky Abstractions" where you end up with variables like `var.aws_specific_setting` inside a generic module.
*   **Result**: Each cloud folder (`aws`, `gcp`, `azure`) uses idiomatic resources for that specific provider.

---

## ⚠️ Important Notes

1.  **State Management**:
    *   By default, these configurations use **Local State** (`terraform.tfstate`).
    *   **Production Advice**: Always configure a remote backend (S3 + DynamoDB for AWS, GCS for GCP, Blob Storage for Azure) to allow team collaboration and locking.

2.  **Kafka Connection**:
    *   These scripts deploy the *Compute* (Kubernetes). They assume *Data* (Kafka) exists externally.
    *   You must configure the `KAFKA_BROKERS` environment variable in your `tfvars` to point to your MSK, Confluent Cloud, or Event Hubs instance.
