# The app itself, shared with the AKS and EKS deployments. Production runs
# at <service>.<domain>; test runs at test.<service>.<domain> against the
# same database, so it doesn't run its own worker.

locals {
  app = {
    image_registry = google_artifact_registry_repository.images.registry_uri

    service_account_annotations = {
      "iam.gke.io/gcp-service-account" = google_service_account.account.email
    }
    node_selector = {
      key    = "cloud.google.com/gke-nodepool"
      values = ["production"]
    }

    ingress_class_name  = "traefik"
    ingress_annotations = { "traefik.ingress.kubernetes.io/router.entrypoints" = "websecure" }
    cluster_issuer      = "letsencrypt-traefik"
    secret_store        = { kind = "ClusterSecretStore", name = "gcp-cluster-store" }

    remote_secrets = {
      mongodb_uri              = google_secret_manager_secret.mongodb_uri.secret_id
      api_key                  = google_secret_manager_secret.api_key.secret_id
      admin_api_key            = google_secret_manager_secret.admin_api_key.secret_id
      xsrf_secret              = google_secret_manager_secret.xsrf_secret.secret_id
      virustotal_api_key       = google_secret_manager_secret.virustotal_api_key.secret_id
      admin_msal_client_id     = google_secret_manager_secret.admin_msal_client_id.secret_id
      admin_msal_client_secret = google_secret_manager_secret.admin_msal_client_secret.secret_id
      msal_client_id           = google_secret_manager_secret.msal_client_id.secret_id
      msal_client_secret       = google_secret_manager_secret.msal_client_secret.secret_id
    }
  }
}

module "prod" {
  source = "../modules/app"

  service     = var.service
  namespace   = "${var.service}-prod"
  host        = "${var.service}.${var.domain}"
  environment = var.environment

  image_registry              = local.app.image_registry
  service_account_annotations = local.app.service_account_annotations
  node_selector               = local.app.node_selector
  ingress_class_name          = local.app.ingress_class_name
  ingress_annotations         = local.app.ingress_annotations
  cluster_issuer              = local.app.cluster_issuer
  tls_secret_name             = "${var.service}-tls"
  secret_store                = local.app.secret_store
  remote_secrets              = local.app.remote_secrets

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
}

module "test" {
  source = "../modules/app"

  service           = var.service
  namespace         = "${var.service}-test"
  host              = "test.${var.service}.${var.domain}"
  environment       = "test"
  autoscaling       = false
  disabled_services = ["worker"]

  image_registry              = local.app.image_registry
  service_account_annotations = local.app.service_account_annotations
  node_selector               = local.app.node_selector
  ingress_class_name          = local.app.ingress_class_name
  ingress_annotations         = local.app.ingress_annotations
  cluster_issuer              = local.app.cluster_issuer
  tls_secret_name             = "${var.service}-test-tls"
  secret_store                = local.app.secret_store
  remote_secrets              = local.app.remote_secrets

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
}

# Resources that lived in this directory before the app moved into the
# module. These blocks keep their state; they can go once applied.

moved {
  from = kubernetes_namespace_v1.prod
  to   = module.prod.kubernetes_namespace_v1.this
}

moved {
  from = kubernetes_service_account_v1.prod
  to   = module.prod.kubernetes_service_account_v1.this
}

moved {
  from = kubernetes_manifest.secrets
  to   = module.prod.kubernetes_manifest.secrets
}

moved {
  from = kubernetes_manifest.certificate
  to   = module.prod.kubernetes_manifest.certificate
}

moved {
  from = kubernetes_ingress_v1.ingress
  to   = module.prod.kubernetes_ingress_v1.this
}

moved {
  from = kubernetes_deployment_v1.api
  to   = module.prod.kubernetes_deployment_v1.service["api"]
}

moved {
  from = kubernetes_service_v1.api
  to   = module.prod.kubernetes_service_v1.service["api"]
}

moved {
  from = kubernetes_deployment_v1.preflight
  to   = module.prod.kubernetes_deployment_v1.service["preflight"]
}

moved {
  from = kubernetes_service_v1.preflight
  to   = module.prod.kubernetes_service_v1.service["preflight"]
}

moved {
  from = kubernetes_deployment_v1.eventupload
  to   = module.prod.kubernetes_deployment_v1.service["eventupload"]
}

moved {
  from = kubernetes_service_v1.eventupload
  to   = module.prod.kubernetes_service_v1.service["eventupload"]
}

moved {
  from = kubernetes_deployment_v1.ruledownload
  to   = module.prod.kubernetes_deployment_v1.service["ruledownload"]
}

moved {
  from = kubernetes_service_v1.ruledownload
  to   = module.prod.kubernetes_service_v1.service["ruledownload"]
}

moved {
  from = kubernetes_deployment_v1.postflight
  to   = module.prod.kubernetes_deployment_v1.service["postflight"]
}

moved {
  from = kubernetes_service_v1.postflight
  to   = module.prod.kubernetes_service_v1.service["postflight"]
}

moved {
  from = kubernetes_deployment_v1.xsrf
  to   = module.prod.kubernetes_deployment_v1.service["xsrf"]
}

moved {
  from = kubernetes_service_v1.xsrf
  to   = module.prod.kubernetes_service_v1.service["xsrf"]
}

moved {
  from = kubernetes_deployment_v1.metrics
  to   = module.prod.kubernetes_deployment_v1.service["metrics"]
}

moved {
  from = kubernetes_service_v1.metrics
  to   = module.prod.kubernetes_service_v1.service["metrics"]
}

moved {
  from = kubernetes_deployment_v1.admin
  to   = module.prod.kubernetes_deployment_v1.service["admin"]
}

moved {
  from = kubernetes_service_v1.admin
  to   = module.prod.kubernetes_service_v1.service["admin"]
}

moved {
  from = kubernetes_deployment_v1.web
  to   = module.prod.kubernetes_deployment_v1.service["web"]
}

moved {
  from = kubernetes_service_v1.web
  to   = module.prod.kubernetes_service_v1.service["web"]
}

moved {
  from = kubernetes_horizontal_pod_autoscaler_v2.preflight
  to   = module.prod.kubernetes_horizontal_pod_autoscaler_v2.service["preflight"]
}

moved {
  from = kubernetes_horizontal_pod_autoscaler_v2.eventupload
  to   = module.prod.kubernetes_horizontal_pod_autoscaler_v2.service["eventupload"]
}

moved {
  from = kubernetes_horizontal_pod_autoscaler_v2.ruledownload
  to   = module.prod.kubernetes_horizontal_pod_autoscaler_v2.service["ruledownload"]
}

moved {
  from = kubernetes_horizontal_pod_autoscaler_v2.postflight
  to   = module.prod.kubernetes_horizontal_pod_autoscaler_v2.service["postflight"]
}

moved {
  from = kubernetes_horizontal_pod_autoscaler_v2.xsrf
  to   = module.prod.kubernetes_horizontal_pod_autoscaler_v2.service["xsrf"]
}

moved {
  from = random_password.admin_secret_key
  to   = module.prod.random_password.flask_secret_key
}

moved {
  from = kubernetes_namespace_v1.test
  to   = module.test.kubernetes_namespace_v1.this
}

moved {
  from = kubernetes_service_account_v1.test
  to   = module.test.kubernetes_service_account_v1.this
}

moved {
  from = kubernetes_manifest.test_secrets
  to   = module.test.kubernetes_manifest.secrets
}

moved {
  from = kubernetes_manifest.test_certificate
  to   = module.test.kubernetes_manifest.certificate
}

moved {
  from = kubernetes_ingress_v1.test_ingress
  to   = module.test.kubernetes_ingress_v1.this
}

moved {
  from = kubernetes_deployment_v1.test_api
  to   = module.test.kubernetes_deployment_v1.service["api"]
}

moved {
  from = kubernetes_service_v1.test_api
  to   = module.test.kubernetes_service_v1.service["api"]
}

moved {
  from = kubernetes_deployment_v1.test_preflight
  to   = module.test.kubernetes_deployment_v1.service["preflight"]
}

moved {
  from = kubernetes_service_v1.test_preflight
  to   = module.test.kubernetes_service_v1.service["preflight"]
}

moved {
  from = kubernetes_deployment_v1.test_eventupload
  to   = module.test.kubernetes_deployment_v1.service["eventupload"]
}

moved {
  from = kubernetes_service_v1.test_eventupload
  to   = module.test.kubernetes_service_v1.service["eventupload"]
}

moved {
  from = kubernetes_deployment_v1.test_ruledownload
  to   = module.test.kubernetes_deployment_v1.service["ruledownload"]
}

moved {
  from = kubernetes_service_v1.test_ruledownload
  to   = module.test.kubernetes_service_v1.service["ruledownload"]
}

moved {
  from = kubernetes_deployment_v1.test_postflight
  to   = module.test.kubernetes_deployment_v1.service["postflight"]
}

moved {
  from = kubernetes_service_v1.test_postflight
  to   = module.test.kubernetes_service_v1.service["postflight"]
}

moved {
  from = kubernetes_deployment_v1.test_xsrf
  to   = module.test.kubernetes_deployment_v1.service["xsrf"]
}

moved {
  from = kubernetes_service_v1.test_xsrf
  to   = module.test.kubernetes_service_v1.service["xsrf"]
}

moved {
  from = kubernetes_deployment_v1.test_metrics
  to   = module.test.kubernetes_deployment_v1.service["metrics"]
}

moved {
  from = kubernetes_service_v1.test_metrics
  to   = module.test.kubernetes_service_v1.service["metrics"]
}

moved {
  from = kubernetes_deployment_v1.test_admin
  to   = module.test.kubernetes_deployment_v1.service["admin"]
}

moved {
  from = kubernetes_service_v1.test_admin
  to   = module.test.kubernetes_service_v1.service["admin"]
}

moved {
  from = kubernetes_deployment_v1.test_web
  to   = module.test.kubernetes_deployment_v1.service["web"]
}

moved {
  from = kubernetes_service_v1.test_web
  to   = module.test.kubernetes_service_v1.service["web"]
}
