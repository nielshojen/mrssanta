variable "project_id" {
  type        = string
  description = "Google Cloud Project ID"
}

variable "region" {
  type        = string
  description = "Google Cloud Region"
}

variable "zone" {
  type        = string
  description = "Google Cloud Region"
}

variable "domain" {
  type = string
}

variable "service" {
  type        = string
  default     = "mrssanta"
  description = "The name of the service"
}

variable "shared_service_name" {
  type        = string
  description = "Name for the shared service"
}

variable "shared_service_k8s_cluster_name" {
  type        = string
  description = "Name for the shared service Kubernetes cluster"
}

variable "environment" {
  type        = string
  description = "Release Environment (eg. dev or prod)"
}

variable "owner" {
  type        = string
  description = "The application owner"
}

variable "team" {
  type        = string
  description = "The application owner team"
}

variable "msal_tenant_id" {
  type        = string
  description = "The MSAL Tenant"
}

variable "admin_msal_client_id" {
  type        = string
  description = "The MSAL Client ID for Admin"
}

variable "msal_client_id" {
  type        = string
  description = "The MSAL Client ID"
}

variable "admin_msal_client_secret" {
  type        = string
  description = "The MSAL Client Secret for Admin"
}

variable "msal_client_secret" {
  type        = string
  description = "The MSAL Client Secret"
}

variable "mongodb_public_key" {
  type        = string
  description = "MongoDB Atlas API Public Key"
}

variable "mongodb_private_key" {
  type        = string
  description = "MongoDB Atlas Private Key"
}

variable "mongodb_project_id" {
  type        = string
  description = "MongoDB Atlas Project ID"
}

variable "service_version" {
  type        = string
  description = "Service version"
  default     = "0.1"
}

variable "sync_interval_monitor" {
  type        = string
  description = "Sync Interval when in Monitor mode (seconds)"
  default     = 14400
}

variable "sync_interval_lockdown" {
  type        = string
  description = "Sync Interval when in Lockdown mode (seconds)"
  default     = 60
}

variable "clean_sync_max_age_days" {
  type        = string
  description = "Days a device may go without a completed clean sync before a full (CLEAN_ALL) sync is forced"
  default     = 30
}

variable "virustotal_api_key" {
  type        = string
  description = "VirusTotal API Key"
}

variable "vote_threshold" {
  type        = string
  description = "Vote Threshold for chaning rules to global"
}

variable "organization" {
  type        = string
  description = "Organization name"
}

variable "support_email" {
  type        = string
  description = "Contact email for support"
}

variable "support_phone" {
  type        = string
  description = "Contact phonenumber for support"
}

variable "labels" {
  type        = map(string)
  description = "The labels to apply to the resources"
}

variable "costcenter" {
  type        = string
  description = "Cost Center for billing purposes"
}

variable "xsrf_required" {
  type        = bool
  description = "Require an XSRF token on sync requests. Santa fetches one from /santa/xsrf/{machine_id} when a request is rejected with 403"
  default     = false
}

variable "logstash_tcp_addr" {
  type        = string
  description = "Optional host:port eventupload also forwards events to"
  default     = "logstash.macgcp.internal:5228"
}

variable "image_registry" {
  type        = string
  description = "Registry the container images are pulled from. The default is the images built by this repository's GitHub workflow; the Artifact Registry repository in artifacts.tf is kept for images you build yourself"
  default     = "ghcr.io/nielshojen/mrssanta"
}

variable "image_tag" {
  type        = string
  description = "Tag of every container image, e.g. a sha-<commit> tag to pin a version"
  default     = "latest"
}
