resource "google_artifact_registry_repository" "images" {
  location      = var.region
  repository_id = "${var.service}-repository"
  description   = "Repository to hold container images ${var.service}"
  format        = "DOCKER"

  vulnerability_scanning_config {
      enablement_config       = "INHERITED"
  }

  labels = "${merge(var.labels, {
    app = "${var.service}"
    service = "${var.service}"
  })}"
}
