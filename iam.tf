resource "google_service_account" "account" {
  account_id   = "${var.service}-sa"
  display_name = "Service Account for ${var.service}"
}

resource "google_project_iam_member" "invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.account.email}"
}

resource "google_project_iam_member" "datastore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.account.email}"
}

resource "google_project_iam_member" "artifactregistryreader" {
  project = var.project_id
  role     = "roles/artifactregistry.reader"
  member   = "serviceAccount:${google_service_account.account.email}"
}

resource "google_project_iam_member" "secretaccessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.account.email}"
}
