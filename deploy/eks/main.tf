# AWS: deploys into an existing EKS cluster that already runs an ingress
# controller, cert-manager and External Secrets Operator. Terraform adds
# Secrets Manager secrets and a Route 53 zone, then deploys the app with the
# shared module. Images come from var.image_registry.

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
    mongodbatlas = {
      source = "mongodb/mongodbatlas"
    }
    random = {
      source = "hashicorp/random"
    }
  }

  # State goes to an S3 bucket you create first. Pass its details at init,
  # see the README.
  backend "s3" {}
}

variable "region" {
  type = string
}

variable "eks_cluster_name" {
  type = string
}

variable "external_secrets_role_name" {
  type        = string
  default     = ""
  description = "IAM role External Secrets Operator uses for the ClusterSecretStore. It is given read access to the app's secrets. Leave empty to grant access yourself"
}

variable "ingress_load_balancer" {
  type = object({
    dns_name = string
    zone_id  = string
  })
  description = "DNS name and hosted zone ID of the load balancer in front of the cluster's ingress controller"
}

variable "tags" {
  type    = map(string)
  default = {}
}

provider "aws" {
  region = var.region

  default_tags {
    tags = merge(var.tags, {
      app        = var.service
      service    = var.service
      env        = var.environment
      owner      = var.owner
      team       = var.team
      costcenter = var.costcenter
    })
  }
}

provider "mongodbatlas" {
  public_key  = var.mongodb_public_key
  private_key = var.mongodb_private_key
}

data "aws_eks_cluster" "this" {
  name = var.eks_cluster_name
}

data "aws_eks_cluster_auth" "this" {
  name = var.eks_cluster_name
}

provider "kubernetes" {
  host                   = data.aws_eks_cluster.this.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.this.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.this.token
}

locals {
  name_servers = aws_route53_zone.this.name_servers
}

# Secrets

resource "aws_secretsmanager_secret" "app" {
  for_each = local.secret_names
  name     = each.value
}

resource "aws_secretsmanager_secret_version" "app" {
  for_each      = local.secret_names
  secret_id     = aws_secretsmanager_secret.app[each.key].id
  secret_string = local.secret_values[each.key]
}

resource "aws_iam_policy" "read_secrets" {
  name        = "${var.service}-read-secrets"
  description = "Read the ${var.service} secrets, for External Secrets Operator"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["secretsmanager:GetSecretValue", "secretsmanager:DescribeSecret"]
      Resource = [for secret in aws_secretsmanager_secret.app : secret.arn]
    }]
  })
}

resource "aws_iam_role_policy_attachment" "external_secrets" {
  count = var.external_secrets_role_name == "" ? 0 : 1

  role       = var.external_secrets_role_name
  policy_arn = aws_iam_policy.read_secrets.arn
}

# DNS

resource "aws_route53_zone" "this" {
  name = "${var.service}.${var.domain}"
}

resource "aws_route53_record" "app" {
  for_each = local.environments

  zone_id = aws_route53_zone.this.zone_id
  name    = each.value.host
  type    = "A"

  alias {
    name                   = var.ingress_load_balancer.dns_name
    zone_id                = var.ingress_load_balancer.zone_id
    evaluate_target_health = false
  }
}
