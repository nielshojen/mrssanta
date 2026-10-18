# App settings shared by every cloud. See the README for what each does.

variable "service" {
  type        = string
  default     = "mrssanta"
  description = "Name prefix for resources and Kubernetes objects, and the MongoDB database name"
}

variable "domain" {
  type        = string
  description = "Parent domain. The app runs at <service>.<domain>, and Terraform creates a DNS zone for that name"
}

variable "environment" {
  type    = string
  default = "prod"
}

variable "enable_test_environment" {
  type        = bool
  default     = false
  description = "Also run a test copy at test.<service>.<domain>, sharing the production database and secrets"
}

variable "image_registry" {
  type        = string
  default     = "ghcr.io/nielshojen/mrssanta"
  description = "Registry the container images are pulled from. The default is the images built by this repository's GitHub workflow"
}

variable "image_tag" {
  type    = string
  default = "latest"
}

variable "owner" {
  type = string
}

variable "team" {
  type = string
}

variable "costcenter" {
  type = string
}

variable "service_version" {
  type    = string
  default = "0.1"
}

# Cluster add-ons the app relies on. They must already be installed.

variable "ingress_class_name" {
  type        = string
  default     = "traefik"
  description = "Ingress class of the cluster's ingress controller"
}

variable "ingress_annotations" {
  type        = map(string)
  default     = { "traefik.ingress.kubernetes.io/router.entrypoints" = "websecure" }
  description = "Annotations for the Ingress"
}

variable "cluster_issuer" {
  type        = string
  description = "cert-manager ClusterIssuer for the TLS certificate"
}

variable "secret_store_name" {
  type        = string
  description = "External Secrets Operator ClusterSecretStore that reads this cloud's secret store"
}

# Sign-in

variable "msal_tenant_id" {
  type        = string
  description = "Entra ID tenant"
}

variable "admin_msal_client_id" {
  type = string
}

variable "admin_msal_client_secret" {
  type      = string
  sensitive = true
}

variable "msal_client_id" {
  type = string
}

variable "msal_client_secret" {
  type      = string
  sensitive = true
}

# MongoDB Atlas

variable "mongodb_public_key" {
  type = string
}

variable "mongodb_private_key" {
  type      = string
  sensitive = true
}

variable "mongodb_project_id" {
  type = string
}

variable "mongodb_cluster_name" {
  type        = string
  description = "Existing Atlas cluster the database lives on"
}

variable "mongodb_uri" {
  type        = string
  sensitive   = true
  description = "Connection string the services use, including the database user's credentials"
}

# App behaviour

variable "virustotal_api_key" {
  type      = string
  sensitive = true
}

variable "sync_interval_lockdown" {
  type    = string
  default = "60"
}

variable "sync_interval_monitor" {
  type    = string
  default = "14400"
}

variable "clean_sync_max_age_days" {
  type    = string
  default = "30"
}

variable "xsrf_required" {
  type    = bool
  default = false
}

variable "vote_threshold" {
  type = string
}

variable "organization" {
  type = string
}

variable "support_email" {
  type = string
}

variable "support_phone" {
  type = string
}

variable "logstash_tcp_addr" {
  type    = string
  default = ""
}
