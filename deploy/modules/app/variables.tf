# Where and how the app runs

variable "service" {
  type        = string
  default     = "mrssanta"
  description = "Name prefix for every Kubernetes object, and the MongoDB database name"
}

variable "namespace" {
  type        = string
  description = "Kubernetes namespace to create and deploy into"
}

variable "host" {
  type        = string
  description = "Public hostname, e.g. mrssanta.example.com. Santa's SyncBaseURL is https://<host>/santa/"
}

variable "environment" {
  type        = string
  description = "Release environment (e.g. prod or test), passed to the event logs"
}

variable "image_registry" {
  type        = string
  description = "Registry the container images are pulled from, without a trailing slash, e.g. 123456789012.dkr.ecr.eu-west-1.amazonaws.com/mrssanta"
}

variable "image_tag" {
  type        = string
  default     = "latest"
  description = "Tag of every container image"
}

variable "autoscaling" {
  type        = bool
  default     = true
  description = "Run extra replicas of the busy sync services and scale them on CPU. Off runs one replica of everything"
}

variable "disabled_services" {
  type        = set(string)
  default     = []
  description = "Services not to deploy, e.g. [\"worker\"] for an environment sharing another one's database"
}

variable "service_account_annotations" {
  type        = map(string)
  default     = {}
  description = "Annotations for the pods' Kubernetes service account, e.g. for workload identity"
}

variable "node_selector" {
  type = object({
    key    = string
    values = list(string)
  })
  default     = null
  description = "Only schedule pods on nodes whose label key has one of these values, e.g. a node pool. Null schedules anywhere"
}

# Cluster add-ons the module relies on

variable "ingress_class_name" {
  type        = string
  description = "Ingress class of the cluster's ingress controller, e.g. traefik or nginx"
}

variable "ingress_annotations" {
  type        = map(string)
  default     = {}
  description = "Annotations for the Ingress, e.g. to select the controller's HTTPS entrypoint"
}

variable "cluster_issuer" {
  type        = string
  description = "cert-manager ClusterIssuer that issues the TLS certificate for host"
}

variable "tls_secret_name" {
  type        = string
  description = "Kubernetes secret cert-manager stores the TLS certificate in"
}

variable "secret_store" {
  type = object({
    kind = string
    name = string
  })
  description = "External Secrets Operator store the secrets are read from, e.g. { kind = \"ClusterSecretStore\", name = \"gcp-cluster-store\" }"
}

variable "remote_secrets" {
  type = object({
    mongodb_uri              = string
    api_key                  = string
    admin_api_key            = string
    xsrf_secret              = string
    virustotal_api_key       = string
    admin_msal_client_id     = string
    admin_msal_client_secret = string
    msal_client_id           = string
    msal_client_secret       = string
  })
  description = "Name of each secret in the secret store"
}

# App settings, described in the README

variable "msal_tenant_id" {
  type        = string
  description = "Entra ID tenant for admin and approval site sign-in"
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
  type        = string
  default     = ""
  description = "Optional host:port eventupload also forwards events to"
}

# Labels on the event logs

variable "service_version" {
  type    = string
  default = "0.1"
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
