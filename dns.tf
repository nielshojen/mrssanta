resource "google_dns_managed_zone" "external" {
  dns_name = "${var.service}.${var.domain}."
  name     = "${var.service}-zone"
  dnssec_config {
    state = "on"
  }
  
  labels = "${merge(var.labels, {
    env = "prod"
    app = "${var.service}"
  })}"
}

resource "google_dns_record_set" "external" {
  managed_zone = google_dns_managed_zone.external.name
  name         = "${var.service}.${var.domain}."
  type         = "A"
  ttl          = "300"
  rrdatas      = [google_compute_global_address.external.address]
}

resource "google_dns_record_set" "dev" {
  managed_zone = google_dns_managed_zone.external.name
  name         = "dev.${var.service}.${var.domain}."
  type         = "A"
  ttl          = "300"
  rrdatas      = ["34.107.69.196"]
}