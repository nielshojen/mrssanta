locals {
  # Path prefix => service. deploy/compose/Caddyfile mirrors this for local
  # runs.
  routes = {
    "/api"                = "api"
    "/santa/preflight"    = "preflight"
    "/santa/eventupload"  = "eventupload"
    "/santa/ruledownload" = "ruledownload"
    "/santa/postflight"   = "postflight"
    "/santa/xsrf"         = "xsrf"
    "/metrics"            = "metrics"
    "/admin"              = "admin"
    "/santa/blockables"   = "web"
    "/santa/moreinfo"     = "web"
    "/santa/token"        = "web"
    "/santa/login"        = "web"
    "/santa/logout"       = "web"
    "/santa/static"       = "web"
  }
}

resource "kubernetes_manifest" "certificate" {
  manifest = {
    apiVersion = "cert-manager.io/v1"
    kind       = "Certificate"
    metadata = {
      name      = var.tls_secret_name
      namespace = kubernetes_namespace_v1.this.metadata[0].name
    }
    spec = {
      secretName = var.tls_secret_name
      issuerRef = {
        name = var.cluster_issuer
        kind = "ClusterIssuer"
      }
      dnsNames = [var.host]
    }
  }
}

resource "kubernetes_ingress_v1" "this" {
  metadata {
    name        = var.service
    namespace   = kubernetes_namespace_v1.this.metadata[0].name
    annotations = var.ingress_annotations
  }

  spec {
    ingress_class_name = var.ingress_class_name

    rule {
      host = var.host

      http {
        dynamic "path" {
          for_each = { for prefix, name in local.routes : prefix => name if contains(keys(local.http_services), name) }
          content {
            path      = path.key
            path_type = "Prefix"

            backend {
              service {
                name = kubernetes_service_v1.service[path.value].metadata[0].name
                port {
                  number = 80
                }
              }
            }
          }
        }
      }
    }

    tls {
      hosts       = [var.host]
      secret_name = var.tls_secret_name
    }
  }
}
