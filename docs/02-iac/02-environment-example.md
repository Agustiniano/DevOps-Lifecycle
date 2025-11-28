# Ejemplo de Environment: Production

## 🏗️ Arquitectura de Producción

```
AWS Region: us-east-1 (primary), us-west-2 (DR)
├── VPC: 10.0.0.0/16
│   ├── Public Subnets: 3 AZs
│   ├── Private Subnets: 3 AZs
│   └── Database Subnets: 3 AZs
│
├── EKS Cluster: prod-eks-cluster
│   ├── Control Plane: Managed by AWS
│   ├── Node Groups:
│   │   ├── General: t3.xlarge (3-10 nodes)
│   │   ├── Compute: c5.2xlarge (2-5 nodes)
│   │   └── Memory: r5.xlarge (2-4 nodes)
│   └── Addons:
│       ├── VPC CNI
│       ├── CoreDNS
│       ├── kube-proxy
│       └── EBS CSI Driver
│
├── RDS PostgreSQL: Multi-AZ, encrypted
├── ElastiCache Redis: Cluster mode enabled
├── S3 Buckets: Versioning + encryption
└── ALB: Internet-facing + Internal
```

## 📁 Production Environment Files

### `environments/production/main.tf`

```hcl
terraform {
  required_version = ">= 1.6.0"

  backend "s3" {
    bucket         = "acme-terraform-state-prod"
    key            = "production/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-state-lock"
    kms_key_id     = "arn:aws:kms:us-east-1:123456789:key/xxx"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11"
    }
  }
}

# Providers
provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Environment = "production"
      Project     = "DevOps-Lifecycle"
      ManagedBy   = "Terraform"
      CostCenter  = "Platform"
      Compliance  = "PCI-DSS"
    }
  }
}

provider "aws" {
  alias  = "dr"
  region = var.dr_region

  default_tags {
    tags = {
      Environment = "production-dr"
      Project     = "DevOps-Lifecycle"
      ManagedBy   = "Terraform"
    }
  }
}

# Data sources
data "aws_availability_zones" "available" {
  state = "available"
}

data "aws_caller_identity" "current" {}

# Local variables
locals {
  cluster_name = "prod-eks-cluster"
  
  common_tags = {
    Environment = "production"
    Project     = "DevOps-Lifecycle"
    ManagedBy   = "Terraform"
  }

  vpc_cidr             = "10.0.0.0/16"
  public_subnet_cidrs  = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  private_subnet_cidrs = ["10.0.10.0/24", "10.0.11.0/24", "10.0.12.0/24"]
  database_subnet_cidrs = ["10.0.20.0/24", "10.0.21.0/24", "10.0.22.0/24"]
}

# ============================================================================
# NETWORKING
# ============================================================================

module "vpc" {
  source = "../../modules/vpc"

  environment    = "production"
  vpc_cidr       = local.vpc_cidr
  cluster_name   = local.cluster_name
  
  public_subnet_cidrs  = local.public_subnet_cidrs
  private_subnet_cidrs = local.private_subnet_cidrs
  
  enable_nat_gateway = true
  enable_flow_logs   = true
  
  flow_logs_retention_days = 90  # Compliance requirement
  
  tags = local.common_tags
}

# ============================================================================
# EKS CLUSTER
# ============================================================================

module "eks" {
  source = "../../modules/eks"

  cluster_name       = local.cluster_name
  kubernetes_version = "1.28"

  vpc_id             = module.vpc.vpc_id
  private_subnet_ids = module.vpc.private_subnet_ids
  public_subnet_ids  = module.vpc.public_subnet_ids

  # Security
  endpoint_public_access = true
  public_access_cidrs    = var.allowed_cidr_blocks  # Office IPs, VPN
  
  # Encryption
  kms_key_arn = aws_kms_key.eks.arn

  # Node groups
  node_groups = {
    general = {
      desired_size = 3
      min_size     = 3
      max_size     = 10
      instance_types = ["t3.xlarge"]
      capacity_type  = "ON_DEMAND"
      disk_size      = 100
      labels = {
        workload = "general"
      }
      taints = []
    }
    
    compute_optimized = {
      desired_size = 2
      min_size     = 2
      max_size     = 5
      instance_types = ["c5.2xlarge"]
      capacity_type  = "ON_DEMAND"
      disk_size      = 100
      labels = {
        workload = "compute-intensive"
      }
      taints = [
        {
          key    = "compute-intensive"
          value  = "true"
          effect = "NoSchedule"
        }
      ]
    }
    
    memory_optimized = {
      desired_size = 2
      min_size     = 1
      max_size     = 4
      instance_types = ["r5.xlarge"]
      capacity_type  = "ON_DEMAND"
      disk_size      = 100
      labels = {
        workload = "memory-intensive"
      }
      taints = [
        {
          key    = "memory-intensive"
          value  = "true"
          effect = "NoSchedule"
        }
      ]
    }
  }

  # EKS Addons
  enable_vpc_cni            = true
  enable_coredns            = true
  enable_kube_proxy         = true
  enable_ebs_csi_driver     = true

  tags = local.common_tags
}

# ============================================================================
# RDS PostgreSQL
# ============================================================================

module "rds_users" {
  source = "../../modules/rds"

  identifier     = "prod-users-db"
  engine         = "postgres"
  engine_version = "15.4"
  instance_class = "db.r6g.xlarge"

  allocated_storage     = 100
  max_allocated_storage = 500
  storage_encrypted     = true
  kms_key_id            = aws_kms_key.rds.arn

  multi_az = true

  database_name = "users"
  master_username = "dbadmin"
  # Password stored in AWS Secrets Manager

  vpc_id             = module.vpc.vpc_id
  subnet_ids         = module.vpc.database_subnet_ids
  
  allowed_security_groups = [module.eks.node_security_group_id]

  backup_retention_period = 30
  backup_window          = "03:00-04:00"
  maintenance_window     = "Mon:04:00-Mon:05:00"

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]

  deletion_protection = true
  skip_final_snapshot = false
  final_snapshot_identifier = "prod-users-db-final-snapshot"

  performance_insights_enabled = true
  performance_insights_retention_period = 7

  tags = merge(
    local.common_tags,
    {
      Component = "database"
      Service   = "user-service"
    }
  )
}

# ============================================================================
# ElastiCache Redis
# ============================================================================

module "redis_cache" {
  source = "../../modules/elasticache"

  cluster_id = "prod-redis-cache"
  engine     = "redis"
  engine_version = "7.0"
  node_type  = "cache.r6g.large"

  num_cache_clusters = 3  # Multi-AZ with automatic failover

  parameter_group_family = "redis7"
  
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token_enabled         = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnet_ids

  allowed_security_groups = [module.eks.node_security_group_id]

  snapshot_retention_limit = 7
  snapshot_window         = "03:00-05:00"
  maintenance_window      = "sun:05:00-sun:07:00"

  tags = merge(
    local.common_tags,
    {
      Component = "cache"
    }
  )
}

# ============================================================================
# S3 Buckets
# ============================================================================

module "s3_application_data" {
  source = "../../modules/s3-bucket"

  bucket_name = "acme-prod-application-data"

  versioning_enabled = true
  
  lifecycle_rules = [
    {
      id      = "transition-to-ia"
      enabled = true
      
      transition = [
        {
          days          = 90
          storage_class = "STANDARD_IA"
        },
        {
          days          = 365
          storage_class = "GLACIER"
        }
      ]
      
      expiration = {
        days = 2555  # 7 years for compliance
      }
    }
  ]

  encryption = {
    sse_algorithm     = "aws:kms"
    kms_master_key_id = aws_kms_key.s3.arn
  }

  block_public_access = true

  tags = merge(
    local.common_tags,
    {
      Component = "storage"
    }
  )
}

# ============================================================================
# Application Load Balancer
# ============================================================================

module "alb_public" {
  source = "../../modules/alb"

  name               = "prod-public-alb"
  load_balancer_type = "application"
  internal           = false

  vpc_id  = module.vpc.vpc_id
  subnets = module.vpc.public_subnet_ids

  enable_deletion_protection = true
  enable_http2              = true
  enable_waf                = true

  security_group_rules = {
    ingress_https = {
      type        = "ingress"
      from_port   = 443
      to_port     = 443
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
      description = "HTTPS from internet"
    }
    ingress_http = {
      type        = "ingress"
      from_port   = 80
      to_port     = 80
      protocol    = "tcp"
      cidr_blocks = ["0.0.0.0/0"]
      description = "HTTP from internet (redirect to HTTPS)"
    }
  }

  access_logs = {
    enabled = true
    bucket  = module.s3_alb_logs.bucket_id
    prefix  = "prod-public-alb"
  }

  tags = local.common_tags
}

# ============================================================================
# KMS Keys
# ============================================================================

resource "aws_kms_key" "eks" {
  description             = "EKS cluster encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = merge(
    local.common_tags,
    {
      Component = "eks"
    }
  )
}

resource "aws_kms_alias" "eks" {
  name          = "alias/prod-eks"
  target_key_id = aws_kms_key.eks.key_id
}

resource "aws_kms_key" "rds" {
  description             = "RDS encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = merge(
    local.common_tags,
    {
      Component = "rds"
    }
  )
}

resource "aws_kms_key" "s3" {
  description             = "S3 bucket encryption key"
  deletion_window_in_days = 30
  enable_key_rotation     = true

  tags = merge(
    local.common_tags,
    {
      Component = "s3"
    }
  )
}

# ============================================================================
# Outputs
# ============================================================================

output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "eks_cluster_endpoint" {
  description = "EKS cluster endpoint"
  value       = module.eks.cluster_endpoint
  sensitive   = true
}

output "eks_cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "rds_endpoint" {
  description = "RDS endpoint"
  value       = module.rds_users.endpoint
  sensitive   = true
}

output "redis_endpoint" {
  description = "Redis endpoint"
  value       = module.redis_cache.primary_endpoint_address
  sensitive   = true
}

output "alb_dns_name" {
  description = "ALB DNS name"
  value       = module.alb_public.dns_name
}
```

### `environments/production/terraform.tfvars`

```hcl
# AWS Configuration
aws_region = "us-east-1"
dr_region  = "us-west-2"

# Network Configuration
allowed_cidr_blocks = [
  "203.0.113.0/24",  # Office Network
  "198.51.100.0/24", # VPN Network
]

# EKS Configuration
kubernetes_version = "1.28"

# Capacity Planning
node_desired_size = 5
node_min_size     = 3
node_max_size     = 20

# Tags
additional_tags = {
  Owner       = "Platform Team"
  Compliance  = "PCI-DSS"
  CostCenter  = "Infrastructure"
  Criticality = "High"
}
```

### `environments/production/variables.tf`

```hcl
variable "aws_region" {
  description = "Primary AWS region"
  type        = string
  default     = "us-east-1"
}

variable "dr_region" {
  description = "Disaster Recovery AWS region"
  type        = string
  default     = "us-west-2"
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to access EKS API"
  type        = list(string)
}

variable "kubernetes_version" {
  description = "Kubernetes version for EKS"
  type        = string
  default     = "1.28"
}

variable "additional_tags" {
  description = "Additional tags for resources"
  type        = map(string)
  default     = {}
}
```

## 🚀 Workflow de Deployment

### 1. Inicialización

```bash
cd infrastructure/terraform/environments/production

# Inicializar Terraform
terraform init

# Validar configuración
terraform validate

# Formatear código
terraform fmt -recursive
```

### 2. Planificación

```bash
# Plan básico
terraform plan -out=tfplan

# Plan con variables específicas
terraform plan \
  -var-file="terraform.tfvars" \
  -out=tfplan

# Review del plan
terraform show tfplan
```

### 3. Aplicación

```bash
# Apply del plan aprobado
terraform apply tfplan

# Apply con auto-approve (SOLO en dev)
terraform apply -auto-approve
```

### 4. Verificación

```bash
# Ver outputs
terraform output

# Ver state
terraform state list

# Inspeccionar recurso específico
terraform state show module.eks.aws_eks_cluster.main
```

## 📊 State Management

### Backend Configuration

```hcl
terraform {
  backend "s3" {
    bucket         = "acme-terraform-state-prod"
    key            = "production/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-state-lock"
    kms_key_id     = "arn:aws:kms:us-east-1:123456789:key/xxx"
  }
}
```

### State Locking con DynamoDB

```bash
# El lock es automático durante apply/plan
# En caso de lock huérfano:
terraform force-unlock <lock-id>
```

### Remote State Data Source

```hcl
data "terraform_remote_state" "network" {
  backend = "s3"
  config = {
    bucket = "acme-terraform-state-prod"
    key    = "network/terraform.tfstate"
    region = "us-east-1"
  }
}

# Usar outputs del otro state
resource "aws_instance" "app" {
  subnet_id = data.terraform_remote_state.network.outputs.private_subnet_ids[0]
}
```

## 🧪 Testing

### Pre-commit Validation

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/antonbabenko/pre-commit-terraform
    rev: v1.83.5
    hooks:
      - id: terraform_fmt
      - id: terraform_validate
      - id: terraform_docs
      - id: terraform_tflint
      - id: terraform_tfsec
      - id: terraform_checkov
```

### TFLint Configuration

```hcl
# .tflint.hcl
plugin "aws" {
  enabled = true
  version = "0.27.0"
  source  = "github.com/terraform-linters/tflint-ruleset-aws"
}

rule "terraform_naming_convention" {
  enabled = true
}

rule "terraform_documented_variables" {
  enabled = true
}
```

### Checkov Scanning

```bash
# Scan de seguridad
checkov -d infrastructure/terraform/environments/production

# Con output en formato JSON
checkov -d infrastructure/terraform/environments/production -o json > checkov-results.json
```

## 🔄 Upgrade & Migration

### Módulo Upgrade

```bash
# Ver versiones disponibles
terraform providers

# Upgrade de providers
terraform init -upgrade

# Test después del upgrade
terraform plan
```

### State Migration

```bash
# Mover recurso en el state
terraform state mv aws_instance.old aws_instance.new

# Importar recurso existente
terraform import aws_instance.web i-1234567890abcdef0

# Remover del state sin destruir
terraform state rm aws_instance.temporary
```
