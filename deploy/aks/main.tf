# Azure: deploys into an existing AKS cluster that already runs an ingress
# controller, cert-manager and External Secrets Operator. Terraform adds a
# resource group with a container registry, a Key Vault for the secrets and
# a DNS zone, then deploys the app with the shared module.

terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
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

  # State goes to an Azure storage account you create first. Pass its
  # details at init, see the README.
  backend "azurerm" {}
}

variable "subscription_id" {
  type = string
}

variable "location" {
  type        = string
  description = "Azure region for the new resources, e.g. westeurope"
}

variable "resource_group_name" {
  type        = string
  default     = "mrssanta"
  description = "Resource group Terraform creates for the registry, Key Vault and DNS zone"
}

variable "aks_cluster_name" {
  type = string
}

variable "aks_resource_group_name" {
  type        = string
  description = "Resource group of the existing AKS cluster"
}

variable "registry_name" {
  type        = string
  description = "Container registry name: globally unique, 5-50 letters and digits"
}

variable "key_vault_name" {
  type        = string
  description = "Key Vault name: globally unique, 3-24 letters, digits and hyphens"
}

variable "external_secrets_principal_id" {
  type        = string
  default     = ""
  description = "Object ID of the identity External Secrets Operator uses for the ClusterSecretStore. It is given read access to the Key Vault's secrets. Leave empty to grant access yourself"
}

variable "ingress_ip" {
  type        = string
  description = "Public IP address of the cluster's ingress controller"
}

variable "tags" {
  type    = map(string)
  default = {}
}

provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
}

provider "mongodbatlas" {
  public_key  = var.mongodb_public_key
  private_key = var.mongodb_private_key
}

data "azurerm_client_config" "current" {}

data "azurerm_kubernetes_cluster" "this" {
  name                = var.aks_cluster_name
  resource_group_name = var.aks_resource_group_name
}

# Uses the cluster's local credentials. For clusters that only allow Entra ID
# sign-in, replace this with an exec block that runs kubelogin.
provider "kubernetes" {
  host                   = data.azurerm_kubernetes_cluster.this.kube_config[0].host
  client_certificate     = base64decode(data.azurerm_kubernetes_cluster.this.kube_config[0].client_certificate)
  client_key             = base64decode(data.azurerm_kubernetes_cluster.this.kube_config[0].client_key)
  cluster_ca_certificate = base64decode(data.azurerm_kubernetes_cluster.this.kube_config[0].cluster_ca_certificate)
}

locals {
  tags = merge(var.tags, {
    app        = var.service
    service    = var.service
    env        = var.environment
    owner      = var.owner
    team       = var.team
    costcenter = var.costcenter
  })

  image_registry = azurerm_container_registry.this.login_server
  name_servers   = azurerm_dns_zone.this.name_servers
}

resource "azurerm_resource_group" "this" {
  name     = var.resource_group_name
  location = var.location
  tags     = local.tags
}

# Images

resource "azurerm_container_registry" "this" {
  name                = var.registry_name
  resource_group_name = azurerm_resource_group.this.name
  location            = azurerm_resource_group.this.location
  sku                 = "Basic"
  admin_enabled       = false
  tags                = local.tags
}

# Lets the cluster's nodes pull the images.
resource "azurerm_role_assignment" "aks_pull" {
  scope                = azurerm_container_registry.this.id
  role_definition_name = "AcrPull"
  principal_id         = data.azurerm_kubernetes_cluster.this.kubelet_identity[0].object_id
}

# Secrets

resource "azurerm_key_vault" "this" {
  name                       = var.key_vault_name
  resource_group_name        = azurerm_resource_group.this.name
  location                   = azurerm_resource_group.this.location
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "standard"
  rbac_authorization_enabled = true
  tags                       = local.tags
}

# Lets Terraform write the secrets.
resource "azurerm_role_assignment" "terraform_secrets" {
  scope                = azurerm_key_vault.this.id
  role_definition_name = "Key Vault Secrets Officer"
  principal_id         = data.azurerm_client_config.current.object_id
}

resource "azurerm_role_assignment" "external_secrets" {
  count = var.external_secrets_principal_id == "" ? 0 : 1

  scope                = azurerm_key_vault.this.id
  role_definition_name = "Key Vault Secrets User"
  principal_id         = var.external_secrets_principal_id
}

resource "azurerm_key_vault_secret" "app" {
  for_each = local.secret_names

  name         = each.value
  value        = local.secret_values[each.key]
  key_vault_id = azurerm_key_vault.this.id
  tags         = local.tags

  depends_on = [azurerm_role_assignment.terraform_secrets]
}

# DNS

resource "azurerm_dns_zone" "this" {
  name                = "${var.service}.${var.domain}"
  resource_group_name = azurerm_resource_group.this.name
  tags                = local.tags
}

resource "azurerm_dns_a_record" "app" {
  for_each = { for name, env in local.environments : name => env }

  name                = each.key == "prod" ? "@" : each.key
  zone_name           = azurerm_dns_zone.this.name
  resource_group_name = azurerm_resource_group.this.name
  ttl                 = 300
  records             = [var.ingress_ip]
  tags                = local.tags
}
