# The app itself, shared with the GKE deployment (deploy/modules/app).

locals {
  secret_names = {
    mongodb_uri              = "${var.service}-mongodb-uri"
    api_key                  = "${var.service}-api-key"
    admin_api_key            = "${var.service}-admin-api-key"
    xsrf_secret              = "${var.service}-xsrf-secret"
    virustotal_api_key       = "${var.service}-virustotal-api-key"
    admin_msal_client_id     = "${var.service}-admin-msal-client-id"
    admin_msal_client_secret = "${var.service}-admin-msal-client-secret"
    msal_client_id           = "${var.service}-msal-client-id"
    msal_client_secret       = "${var.service}-msal-client-secret"
  }

  secret_values = {
    mongodb_uri              = var.mongodb_uri
    api_key                  = random_password.api_key.result
    admin_api_key            = random_password.admin_api_key.result
    xsrf_secret              = random_password.xsrf_secret.result
    virustotal_api_key       = var.virustotal_api_key
    admin_msal_client_id     = var.admin_msal_client_id
    admin_msal_client_secret = var.admin_msal_client_secret
    msal_client_id           = var.msal_client_id
    msal_client_secret       = var.msal_client_secret
  }

  environments = merge(
    { prod = { host = "${var.service}.${var.domain}", tls = "${var.service}-tls" } },
    var.enable_test_environment ? { test = { host = "test.${var.service}.${var.domain}", tls = "${var.service}-test-tls" } } : {},
  )
}

# Santa and API clients send these in the X-API-Key header.
resource "random_password" "api_key" {
  length  = 64
  special = false
}

resource "random_password" "admin_api_key" {
  length  = 64
  special = false
}

resource "random_password" "xsrf_secret" {
  length  = 64
  special = false
}

module "atlas" {
  source = "../modules/atlas"

  project_id   = var.mongodb_project_id
  cluster_name = var.mongodb_cluster_name
  database     = var.service
}

module "app" {
  source   = "../modules/app"
  for_each = local.environments

  service     = var.service
  namespace   = "${var.service}-${each.key}"
  host        = each.value.host
  environment = each.key == "prod" ? var.environment : each.key

  # The test copy shares production's database, so it runs no background
  # jobs and no extra replicas.
  autoscaling       = each.key == "prod"
  disabled_services = each.key == "prod" ? [] : ["worker"]

  image_registry      = var.image_registry
  image_tag           = var.image_tag
  ingress_class_name  = var.ingress_class_name
  ingress_annotations = var.ingress_annotations
  cluster_issuer      = var.cluster_issuer
  tls_secret_name     = each.value.tls
  secret_store        = { kind = "ClusterSecretStore", name = var.secret_store_name }
  remote_secrets      = local.secret_names

  msal_tenant_id          = var.msal_tenant_id
  sync_interval_lockdown  = var.sync_interval_lockdown
  sync_interval_monitor   = var.sync_interval_monitor
  clean_sync_max_age_days = var.clean_sync_max_age_days
  xsrf_required           = var.xsrf_required
  vote_threshold          = var.vote_threshold
  organization            = var.organization
  support_email           = var.support_email
  support_phone           = var.support_phone
  logstash_tcp_addr       = var.logstash_tcp_addr
  service_version         = var.service_version
  owner                   = var.owner
  team                    = var.team
  costcenter              = var.costcenter

  # Secrets must exist in the store before External Secrets Operator reads them.
  depends_on = [aws_secretsmanager_secret_version.app]
}

output "sync_base_url" {
  value       = module.app["prod"].sync_base_url
  description = "Santa's SyncBaseURL"
}


output "name_servers" {
  value       = local.name_servers
  description = "Delegate <service>.<domain> to these name servers"
}
