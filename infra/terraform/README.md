# Infrastructure Deployment (Azure)

This directory contains the Terraform configuration to deploy the Go Event Ingestor to Azure Kubernetes Service (AKS).

## 🏗 Architecture

Resources created:
1.  **Resource Group**: Container for all resources.
2.  **Azure Container Registry (ACR)**: To store the application Docker images.
3.  **AKS Cluster**: Single node pool cluster to run the application.
    *   System-assigned Identity enabled.
    *   Log Analytics integrated for monitoring.
4.  **Role Assignment**: Grants AKS (kubelet) permission to pull images from ACR.
5.  **Kubernetes Resources**:
    *   **Namespace**: `ingestor`
    *   **Deployment**: Runs the Go app with configured limits and environment variables.
    *   **Service**: LoadBalancer exposing the app on port 80.

## 📋 Prerequisites

*   [Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)
*   [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.5
*   `kubectl`

## 🚀 Deployment Steps

### 1. Authenticate with Azure

```bash
az login
```

### 2. Initialize Terraform

```bash
cd infra/terraform
terraform init
```

### 3. Plan and Apply Infrastructure

Create a `terraform.tfvars` file (or use defaults):

```bash
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your specific Kafka details
```

Run the plan:
```bash
terraform plan -out=tfplan
```

Apply the infrastructure (this takes 10-15 minutes):
```bash
terraform apply tfplan
```

### 4. Build and Push Image

Once Terraform finishes, you will get the ACR login server in the outputs. You need to build and push the image so the Kubernetes deployment can pull it.

```bash
# Get ACR name from output
ACR_SERVER=$(terraform output -raw acr_login_server)
az acr login --name ${ACR_SERVER%%.*} # Log in to ACR

# Go back to root
cd ../..

# Build and Tag
docker build -t $ACR_SERVER/go-event-ingestor:latest .

# Push
docker push $ACR_SERVER/go-event-ingestor:latest
```

### 5. Verify Deployment

Connect to the cluster:
```bash
# Run the command provided in terraform output
$(terraform output -raw get_credentials_command)
```

Check the pods:
```bash
kubectl get pods -n ingestor
```

Get the Public IP:
```bash
kubectl get svc -n ingestor
```

## 🧹 Cleanup

To destroy all resources:
```bash
terraform destroy
```

## ⚠️ Notes

*   **Kafka**: This setup assumes you have an external Kafka provider (e.g., Confluent Cloud). You must configure `KAFKA_BROKERS` in `terraform.tfvars`.
*   **State**: By default, Terraform state is local. For production, configure a remote backend (Azure Storage Account) in `versions.tf`.
