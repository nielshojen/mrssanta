resource "google_cloud_run_service" "api" {
  name                       = "${var.service}-api"
  location                   = var.region
  autogenerate_revision_name = true

  template {
    spec {
      service_account_name = google_service_account.account.email
      containers {
        image = "${google_artifact_registry_repository.images.location}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.repository_id}/${var.service}-api:${var.service_version}"

        env {
          name = "GCP_PROJECT"
          value = var.project_id
        }

        env {
          name = "DB_PREFIX"
          value = var.service
        }
      }
    }

    metadata {
      labels = {
        env = "${var.environment}"
        app = "${var.service}"
        service = "${var.environment}"
        owner = "${var.owner}"
        team = "${var.team}"
        version = replace(var.service_version, ".", "-"),
      }
      annotations = {
        "autoscaling.knative.dev/minScale"        = "0"
        "autoscaling.knative.dev/maxScale"        = "2"
        "run.googleapis.com/client-name"          = "terraform"
        "run.googleapis.com/cpu-throttling"       = "true"
        "run.googleapis.com/startup-cpu-boost"    = "true"
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}