# resource "google_cloud_run_service" "gui" {
#   name     = "${var.service}-gui"
#   location = var.region

#   metadata {
#     # annotations = {
#     #   "run.googleapis.com/ingress"        = "internal-and-cloud-load-balancing"
#     #   "run.googleapis.com/ingress-status" = "internal-and-cloud-load-balancing"
#     # }
#     labels = "${merge(var.labels, {
#       env = "${var.environment}"
#       app = "${var.service}"
#       service = "${var.environment}"
#       owner = "${var.owner}"
#       team = "${var.team}"
#       version = replace(var.service_version, ".", "-"),
#     })}"
#   }

#     template {
#         spec {
#             service_account_name = google_service_account.account.email
#             containers {
#                 image = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.images.name}/${var.service}-gui:latest"
#                 ports {
#                     container_port = 8080
#                 }
#                 env {
#                     name  = "GOOGLE_CLOUD_PROJECT"
#                     value = var.project_id
#                 }
#                 env {
#                     name  = "FIRESTORE_DATABASE"
#                     value = google_firestore_database.database.name
#                 }
#             }
#         }

#     metadata {
#       labels = {
#         app = "${var.service}"
#         service = "${var.environment}"
#         env = "${var.environment}"
#         version = replace(var.service_version, ".", "-"),
#       }
#       annotations = {
#         "autoscaling.knative.dev/minScale"        = "0"
#         "autoscaling.knative.dev/maxScale"        = "100"
#         # "run.googleapis.com/vpc-access-connector" = "projects/workplace-f488/locations/europe-west3/connectors/serverless-connector"
#         # "run.googleapis.com/vpc-access-egress"    = "private-ranges-only"
#         "run.googleapis.com/client-name"          = "terraform"
#         "run.googleapis.com/cpu-throttling"       = "true"
#         "run.googleapis.com/startup-cpu-boost"    = "true"
#       }
#     }
#     }
# }

# resource "google_cloud_run_service_iam_policy" "noauth" {
#     location = google_cloud_run_service.gui.location
#     service  = google_cloud_run_service.gui.name

#     policy_data = jsonencode({
#         bindings = [{
#         role = "roles/run.invoker"
#         members = ["allUsers"]
#         }]
#     })
# }
