data "aws_availability_zones" "available" {}

# 1. Network (VPC)
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "${var.cluster_name}-vpc"
  cidr = "10.0.0.0/16"

  azs             = slice(data.aws_availability_zones.available.names, 0, 2)
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24"]

  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true

  tags = {
    Project     = "GoEventIngestor"
    Environment = "Production"
  }
}

# 2. EKS Cluster
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 19.0"

  cluster_name    = var.cluster_name
  cluster_version = "1.29"

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets

  cluster_endpoint_public_access = true

  eks_managed_node_group_defaults = {
    ami_type = "AL2_x86_64"
  }

  eks_managed_node_groups = {
    one = {
      name = "node-group-1"

      instance_types = ["t3.small"]

      min_size     = 1
      max_size     = 3
      desired_size = 2
    }
  }
}

# 3. ECR Repository
resource "aws_ecr_repository" "repo" {
  name                 = var.image_repository
  image_tag_mutability = "MUTABLE"
  force_delete         = true
}

# 4. Kubernetes Provider Config
provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)

  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    # This requires the awscli to be installed locally where Terraform is executed
    args = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
  }
}

# 5. Application Deployment (Shared Module)
module "app" {
  source = "../modules/k8s-app"

  namespace = "ingestor"
  app_name  = "go-event-ingestor"
  # ECR URL format: AWS_ACCOUNT_ID.dkr.ecr.REGION.amazonaws.com/REPO:TAG
  image     = "${aws_ecr_repository.repo.repository_url}:${var.image_tag}"
  replicas  = var.app_replicas
  env_vars  = var.app_env
  port      = 8080

  depends_on = [module.eks]
}
