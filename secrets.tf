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

# API Key
resource "random_password" "api_key" {
  length           = 64
  special          = false
}

resource "google_secret_manager_secret" "api_key" {
  secret_id = "${var.service}-api-key"

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

resource "google_secret_manager_secret_version" "api_key" {
  secret      = google_secret_manager_secret.api_key.id
  secret_data = random_password.api_key.result
}

resource "google_secret_manager_secret_iam_binding" "api_key" {

  secret_id = google_secret_manager_secret.api_key.id
  role      = "roles/secretmanager.secretAccessor"
  members   = ["serviceAccount:${google_service_account.account.email}"]
}

# CLIENT_ID Key
resource "random_password" "client_id" {
  length           = 32
  special          = false
}

resource "google_secret_manager_secret" "client_id" {
  secret_id = "${var.service}-msal-client-id"

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

resource "google_secret_manager_secret_version" "client_id" {
  secret      = google_secret_manager_secret.client_id.id
  secret_data = var.client_id
}

resource "google_secret_manager_secret_iam_binding" "client_id" {

  secret_id = google_secret_manager_secret.client_id.id
  role      = "roles/secretmanager.secretAccessor"
  members   = ["serviceAccount:${google_service_account.account.email}"]
}

# TENANT_ID Key
resource "random_password" "tenant_id" {
  length           = 32
  special          = false
}

resource "google_secret_manager_secret" "tenant_id" {
  secret_id = "${var.service}-msal-tenant-id"

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

resource "google_secret_manager_secret_version" "tenant_id" {
  secret      = google_secret_manager_secret.tenant_id.id
  secret_data = var.tenant_id
}

resource "google_secret_manager_secret_iam_binding" "tenant_id" {

  secret_id = google_secret_manager_secret.tenant_id.id
  role      = "roles/secretmanager.secretAccessor"
  members   = ["serviceAccount:${google_service_account.account.email}"]
}