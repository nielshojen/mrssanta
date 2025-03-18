resource "random_password" "gui_secret_key" {
  length           = 64
  special          = false
}

resource "google_cloud_run_service" "gui" {
  name     = "${var.service}-gui"
  location = var.region

  metadata {
    # annotations = {
    #   "run.googleapis.com/ingress"        = "internal-and-cloud-load-balancing"
    #   "run.googleapis.com/ingress-status" = "internal-and-cloud-load-balancing"
    # }
    labels = "${merge(var.labels, {
      env = "${var.environment}"
      app = "${var.service}"
      service = "${var.environment}"
      owner = "${var.owner}"
      team = "${var.team}"
      version = replace(var.service_version, ".", "-"),
    })}"
  }

    template {
        spec {
            service_account_name = google_service_account.account.email
            containers {
                image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.name}/${var.service}-gui:latest"

                ports {
                    container_port = 8080
                }

                env {
                    name  = "FLASK_SECRET_KEY"
                    value = random_password.gui_secret_key.result
                }

                env {
                    name  = "GOOGLE_CLOUD_PROJECT"
                    value = var.project_id
                }

                env {
                    name  = "FIRESTORE_DATABASE"
                    value = google_firestore_database.database.name
                }

                env {
                    name = "MONGO_URI"
                    value_from {
                        secret_key_ref {
                            name = google_secret_manager_secret.mongodb_uri.secret_id
                            key  = "latest"
                        }
                    }
                }

                env {
                    name = "VT_API_KEY"
                    value_from {
                        secret_key_ref {
                            name = google_secret_manager_secret.virustotal_api_key.secret_id
                            key  = "latest"
                        }
                    }
                }

                env {
                    name  = "MSAL_TENANT_ID"
                    value = var.msal_tenant_id
                }

                env {
                    name  = "MSAL_REDIRECT_URI"
                    value = "https://mrssanta-gui-3mfoapoj6a-ey.a.run.app/token"
                }

                env {
                    name = "MSAL_CLIENT_ID"
                    value_from {
                        secret_key_ref {
                            name = google_secret_manager_secret.msal_client_id.secret_id
                            key  = "latest"
                        }
                    }
                }

                env {
                    name = "MSAL_CLIENT_SECRET"
                    value_from {
                        secret_key_ref {
                            name = google_secret_manager_secret.msal_client_secret.secret_id
                            key  = "latest"
                        }
                    }
                }
            }
        }

    metadata {
      labels = {
        app = "${var.service}"
        service = "${var.environment}"
        env = "${var.environment}"
        version = replace(var.service_version, ".", "-"),
      }
      annotations = {
        "autoscaling.knative.dev/minScale"        = "0"
        "autoscaling.knative.dev/maxScale"        = "100"
        "run.googleapis.com/vpc-access-connector" = "projects/workplace-f488/locations/europe-west3/connectors/serverless-connector"
        "run.googleapis.com/vpc-access-egress"    = "private-ranges-only"
        "run.googleapis.com/client-name"          = "terraform"
        "run.googleapis.com/cpu-throttling"       = "true"
        "run.googleapis.com/startup-cpu-boost"    = "true"
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
