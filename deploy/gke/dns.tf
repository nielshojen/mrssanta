resource "google_dns_managed_zone" "external" {
  dns_name = "${var.service}.${var.domain}."
  name     = "${var.service}-zone"
  dnssec_config {
    state = "on"
  }
  
  labels = "${merge(var.labels, {
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.service}"
    owner = "${var.owner}"
    team = "${var.team}"
  })}"
}

resource "google_dns_record_set" "external" {
  managed_zone = google_dns_managed_zone.external.name
  name         = "${var.service}.${var.domain}."
  type         = "A"
  ttl          = "300"
  rrdatas      = [data.google_compute_address.ingress_ip.address]
}

resource "google_dns_record_set" "test" {
  managed_zone = google_dns_managed_zone.external.name
  name         = "test.${var.service}.${var.domain}."
  type         = "A"
  ttl          = "300"
  rrdatas      = [data.google_compute_address.ingress_ip.address]
}