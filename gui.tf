resource "google_cloud_run_service" "gui" {
  name     = "${var.service}-gui"
  location = var.region

    template {
        spec {
            service_account_name = google_service_account.account.email
            containers {
                image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.name}/${var.service}-gui:latest"
                ports {
                    container_port = 8080
                }
                env {
                    name  = "GOOGLE_CLOUD_PROJECT"
                    value = var.project_id
                }
                env {
                    name  = "FIRESTORE_DATABASE"
                    value = google_firestore_database.database.name
                }
            }
        }
    }
}

resource "google_cloud_run_service_iam_policy" "noauth" {
    location = google_cloud_run_service.gui.location
    service  = google_cloud_run_service.gui.name

    policy_data = jsonencode({
        bindings = [{
        role = "roles/run.invoker"
        members = ["allUsers"]
        }]
    })
}
