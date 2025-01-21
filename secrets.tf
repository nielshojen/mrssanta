resource "google_secret_manager_secret" "virustotal_api_key" {
  secret_id = "${var.service}-virustotal-api-key"

  labels = "${merge(var.labels, {
      app     = "${var.service}",
      service = "${var.service}",
      env     = "prod",
  })}"

  replication {
    auto {}
  }

  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "virustotal_api_key" {
  secret      = google_secret_manager_secret.virustotal_api_key.id
  secret_data = var.virustotal_api_key
}

resource "google_secret_manager_secret_iam_binding" "virustotal_api_key" {

  secret_id = google_secret_manager_secret.virustotal_api_key.id
  role      = "roles/secretmanager.secretAccessor"
  members   = ["serviceAccount:${google_service_account.account.email}"]
}