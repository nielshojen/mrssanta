terraform {
  required_providers {
    kubernetes = {
      source = "hashicorp/kubernetes"
    }
    random = {
      source = "hashicorp/random"
    }
  }
}

locals {
  secrets_name = "${var.service}-secrets"

  sync_env = {
    MONGO_DB      = var.service
    XSRF_REQUIRED = tostring(var.xsrf_required)
  }
  sync_secrets = {
    MONGO_URI   = "mongodb_uri"
    API_KEY     = "api_key"
    XSRF_SECRET = "xsrf_secret"
  }

  default_limits   = { cpu = "1000m", memory = "2Gi" }
  default_requests = { cpu = "50m", memory = "64Mi" }

  # Every container. env holds plain values; secret_env maps a variable to a
  # key of var.remote_secrets. replicas and autoscale apply when
  # var.autoscaling is on. Services with http = false get no Service.
  all_services = {
    api = {
      http       = true
      replicas   = 1
      autoscale  = null
      limits     = local.default_limits
      requests   = local.default_requests
      env        = { MONGO_DB = var.service }
      secret_env = { MONGO_URI = "mongodb_uri", API_KEY = "admin_api_key" }
    }
    preflight = {
      http      = true
      replicas  = 1
      autoscale = { min = 1, max = 3 }
      limits    = local.default_limits
      requests  = local.default_requests
      env = merge(local.sync_env, {
        SYNC_INTERVAL_LOCKDOWN  = var.sync_interval_lockdown
        SYNC_INTERVAL_MONITOR   = var.sync_interval_monitor
        CLEAN_SYNC_MAX_AGE_DAYS = var.clean_sync_max_age_days
      })
      secret_env = local.sync_secrets
    }
    eventupload = {
      http      = true
      replicas  = 2
      autoscale = { min = 2, max = 6 }
      limits    = local.default_limits
      requests  = local.default_requests
      env = merge(local.sync_env, {
        LOGSTASH_TCP_ADDR = var.logstash_tcp_addr
        DD_SOURCE         = var.service
        DD_SERVICE        = "${var.service}-eventupload"
        DD_ENV            = var.environment
        COSTCENTER        = var.costcenter
        VERSION           = var.service_version
        TEAM              = var.team
        OWNER             = var.owner
        ENVIRONMENT       = var.environment
      })
      secret_env = local.sync_secrets
    }
    ruledownload = {
      http       = true
      replicas   = 2
      autoscale  = { min = 2, max = 6 }
      limits     = local.default_limits
      requests   = local.default_requests
      env        = merge(local.sync_env, { BATCH_SIZE = "1000" })
      secret_env = local.sync_secrets
    }
    postflight = {
      http       = true
      replicas   = 1
      autoscale  = { min = 1, max = 3 }
      limits     = local.default_limits
      requests   = local.default_requests
      env        = local.sync_env
      secret_env = local.sync_secrets
    }
    xsrf = {
      http       = true
      replicas   = 1
      autoscale  = { min = 1, max = 3 }
      limits     = { cpu = "250m", memory = "128Mi" }
      requests   = local.default_requests
      env        = {}
      secret_env = { API_KEY = "api_key", XSRF_SECRET = "xsrf_secret" }
    }
    metrics = {
      http       = true
      replicas   = 1
      autoscale  = null
      limits     = { cpu = "500m", memory = "256Mi" }
      requests   = local.default_requests
      env        = {}
      secret_env = {}
    }
    admin = {
      http      = true
      replicas  = 1
      autoscale = null
      limits    = local.default_limits
      requests  = { cpu = "50m", memory = "128Mi" }
      env = {
        FLASK_SECRET_KEY        = random_password.flask_secret_key.result
        CLEAN_SYNC_MAX_AGE_DAYS = var.clean_sync_max_age_days
        MSAL_TENANT_ID          = var.msal_tenant_id
        MSAL_REDIRECT_URI       = "https://${var.host}/admin/token"
      }
      secret_env = {
        MONGO_URI          = "mongodb_uri"
        VT_API_KEY         = "virustotal_api_key"
        MSAL_CLIENT_ID     = "admin_msal_client_id"
        MSAL_CLIENT_SECRET = "admin_msal_client_secret"
      }
    }
    web = {
      http      = true
      replicas  = 1
      autoscale = null
      limits    = local.default_limits
      requests  = { cpu = "50m", memory = "128Mi" }
      env = {
        FLASK_SECRET_KEY       = random_password.flask_secret_key.result
        ORGANIZATION           = var.organization
        SUPPORT_EMAIL          = var.support_email
        SUPPORT_PHONE          = var.support_phone
        VOTE_THRESHOLD         = var.vote_threshold
        SYNC_INTERVAL_LOCKDOWN = var.sync_interval_lockdown
        MONGO_DB               = var.service
        MSAL_TENANT_ID         = var.msal_tenant_id
        MSAL_REDIRECT_URI      = "https://${var.host}/santa/token"
      }
      secret_env = {
        MONGO_URI          = "mongodb_uri"
        VT_API_KEY         = "virustotal_api_key"
        MSAL_CLIENT_ID     = "msal_client_id"
        MSAL_CLIENT_SECRET = "msal_client_secret"
      }
    }
    # Background jobs. One replica, replaced rather than rolled, so jobs
    # never run twice. The MalwareBazaar job spools its download to /tmp.
    worker = {
      http       = false
      replicas   = 1
      autoscale  = null
      limits     = { cpu = "250m", memory = "256Mi", "ephemeral-storage" = "5Gi" }
      requests   = { cpu = "10m", memory = "32Mi", "ephemeral-storage" = "1Gi" }
      env        = { MONGO_DB = var.service }
      secret_env = { MONGO_URI = "mongodb_uri" }
    }
  }

  services      = { for name, s in local.all_services : name => s if !contains(var.disabled_services, name) }
  http_services = { for name, s in local.services : name => s if s.http }
  autoscaled    = var.autoscaling ? { for name, s in local.services : name => s if s.autoscale != null } : {}
}

resource "kubernetes_namespace_v1" "this" {
  metadata {
    name = var.namespace
    annotations = {
      name = var.namespace
    }
  }
}

resource "kubernetes_service_account_v1" "this" {
  metadata {
    name        = "${var.service}-ksa"
    namespace   = kubernetes_namespace_v1.this.metadata[0].name
    annotations = var.service_account_annotations
  }
}

# Signs the admin and approval site session cookies.
resource "random_password" "flask_secret_key" {
  length  = 64
  special = false
}

# Copies the secrets from the cloud's secret store into one Kubernetes
# secret. Each key is the secret's name in the store.
resource "kubernetes_manifest" "secrets" {
  manifest = {
    apiVersion = "external-secrets.io/v1"
    kind       = "ExternalSecret"
    metadata = {
      name      = local.secrets_name
      namespace = kubernetes_namespace_v1.this.metadata[0].name
    }
    spec = {
      refreshInterval = "5m0s"
      secretStoreRef  = var.secret_store
      target = {
        name           = local.secrets_name
        creationPolicy = "Owner"
      }
      data = [
        for name in sort(values(var.remote_secrets)) : {
          secretKey = name
          remoteRef = { key = name }
        }
      ]
    }
  }
}

resource "kubernetes_deployment_v1" "service" {
  for_each = local.services

  metadata {
    name      = "${var.service}-${each.key}"
    namespace = kubernetes_namespace_v1.this.metadata[0].name
    labels = {
      app = "${var.service}-${each.key}"
    }
  }

  spec {
    replicas = var.autoscaling ? each.value.replicas : 1

    dynamic "strategy" {
      for_each = each.value.http ? [] : [1]
      content {
        type = "Recreate"
      }
    }

    selector {
      match_labels = {
        app = "${var.service}-${each.key}"
      }
    }

    template {
      metadata {
        labels = {
          app = "${var.service}-${each.key}"
        }
      }

      spec {
        affinity {
          dynamic "node_affinity" {
            for_each = var.node_selector == null ? [] : [var.node_selector]
            content {
              required_during_scheduling_ignored_during_execution {
                node_selector_term {
                  match_expressions {
                    key      = node_affinity.value.key
                    operator = "In"
                    values   = node_affinity.value.values
                  }
                }
              }
            }
          }

          # Spread replicas over nodes.
          pod_anti_affinity {
            preferred_during_scheduling_ignored_during_execution {
              weight = 100
              pod_affinity_term {
                label_selector {
                  match_expressions {
                    key      = "app"
                    operator = "In"
                    values   = ["${var.service}-${each.key}"]
                  }
                }
                topology_key = "kubernetes.io/hostname"
              }
            }
          }
        }

        service_account_name = kubernetes_service_account_v1.this.metadata[0].name

        container {
          name              = each.key
          image             = "${var.image_registry}/${each.key}:${var.image_tag}"
          image_pull_policy = "Always"

          dynamic "port" {
            for_each = each.value.http ? [8080] : []
            content {
              container_port = port.value
            }
          }

          resources {
            limits   = each.value.limits
            requests = each.value.requests
          }

          dynamic "env" {
            for_each = each.value.env
            content {
              name  = env.key
              value = env.value
            }
          }

          dynamic "env" {
            for_each = each.value.secret_env
            content {
              name = env.key
              value_from {
                secret_key_ref {
                  name = local.secrets_name
                  key  = var.remote_secrets[env.value]
                }
              }
            }
          }
        }
      }
    }
  }

  depends_on = [kubernetes_manifest.secrets]
}

resource "kubernetes_service_v1" "service" {
  for_each = local.http_services

  metadata {
    name      = "${var.service}-${each.key}"
    namespace = kubernetes_namespace_v1.this.metadata[0].name
    labels = {
      app = "${var.service}-${each.key}"
    }
    annotations = {
      # Only the Ingress is exposed; on GKE, skip standalone network endpoint groups.
      "cloud.google.com/neg" = jsonencode({ ingress = false })
    }
  }

  spec {
    selector = {
      app = "${var.service}-${each.key}"
    }

    port {
      name        = "http"
      port        = 80
      target_port = 8080
      protocol    = "TCP"
    }

    type = "ClusterIP"
  }
}

resource "kubernetes_horizontal_pod_autoscaler_v2" "service" {
  for_each = local.autoscaled

  metadata {
    name      = "${var.service}-${each.key}"
    namespace = kubernetes_namespace_v1.this.metadata[0].name
  }

  spec {
    min_replicas = each.value.autoscale.min
    max_replicas = each.value.autoscale.max

    scale_target_ref {
      api_version = "apps/v1"
      kind        = "Deployment"
      name        = kubernetes_deployment_v1.service[each.key].metadata[0].name
    }

    metric {
      type = "Resource"
      resource {
        name = "cpu"
        target {
          type                = "Utilization"
          average_utilization = 70
        }
      }
    }
  }
}
