resource "google_compute_region_network_endpoint_group" "backend" {
  provider              = google-beta
  project               = var.project_id
  name                  = "${var.project_id}-${var.service}-endpoint-group"
  region                = var.region
  network_endpoint_type = "SERVERLESS"
  serverless_deployment {
    platform = "apigateway.googleapis.com"
    resource = google_api_gateway_gateway.gw.gateway_id
  }
}

resource "google_compute_global_address" "external" {
  provider = google-beta

  name = "${var.service}-address"

  labels = "${merge(var.labels, {
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.environment}"
    owner = "${var.owner}"
    team = "${var.team}"
  })}"
}