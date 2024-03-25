resource "google_compute_managed_ssl_certificate" "external" {
  name = "${var.service}-cert"

  managed {
     domains = ["${var.service}.${var.domain}"]
  }
}

resource "google_compute_backend_service" "backend" {
  name      = "${var.service}-backend"

  custom_response_headers = ["X-Robots-Tag: noindex, nofollow"]

  protocol  = "HTTP"
  port_name = "http"
  timeout_sec = 30

  backend {
    group = google_compute_region_network_endpoint_group.backend.id
  }
}

resource "google_compute_url_map" "urlmap" {
  name            = "${var.service}-urlmap"

  default_service = google_compute_backend_service.backend.id

  host_rule {
    hosts        = ["${var.service}.${var.domain}"]
    path_matcher = "allpaths"
  }

  path_matcher {
    name = "allpaths"
    default_service = google_compute_backend_service.backend.id
  }
}

resource "google_compute_target_https_proxy" "backend" {
  name   = "${var.service}-https-proxy"

  url_map          = google_compute_url_map.urlmap.id
  ssl_certificates = [
    google_compute_managed_ssl_certificate.external.id
  ]
}

resource "google_compute_global_forwarding_rule" "backend" {
  name   = "${var.service}-lb"

  target = google_compute_target_https_proxy.backend.id
  port_range = "443"
  ip_address = google_compute_global_address.external.address
}

resource "google_compute_url_map" "https_redirect" {
  name            = "${var.service}-https-redirect"

  default_url_redirect {
    https_redirect         = true
    redirect_response_code = "MOVED_PERMANENTLY_DEFAULT"
    strip_query            = false
  }
}

resource "google_compute_target_http_proxy" "https_redirect" {
  name    = "${var.service}-http-proxy"
  url_map = google_compute_url_map.https_redirect.id
}

resource "google_compute_global_forwarding_rule" "https_redirect" {
  name   = "${var.service}-lb-http"

  target = google_compute_target_http_proxy.https_redirect.id
  port_range = "80"
  ip_address = google_compute_global_address.external.address
}