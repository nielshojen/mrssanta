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

# MongoDB URI
resource "google_secret_manager_secret" "mongodb_uri" {
  secret_id = "${var.service}-mongodb-uri"

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

resource "google_secret_manager_secret_iam_binding" "mongodb_uri" {

  secret_id = google_secret_manager_secret.mongodb_uri.id
  role      = "roles/secretmanager.secretAccessor"
  members   = ["serviceAccount:${google_service_account.account.email}"]
}

# MSAL Client ID
resource "google_secret_manager_secret" "msal_client_id" {
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

resource "google_secret_manager_secret_version" "msal_client_id" {
  secret      = google_secret_manager_secret.msal_client_id.id
  secret_data = var.msal_client_id
}

resource "google_secret_manager_secret_iam_binding" "msal_client_id" {

  secret_id = google_secret_manager_secret.msal_client_id.id
  role      = "roles/secretmanager.secretAccessor"
  members   = ["serviceAccount:${google_service_account.account.email}"]
}

# MSAL Client Secret
resource "google_secret_manager_secret" "msal_client_secret" {
  secret_id = "${var.service}-msal-client-secret"

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

resource "google_secret_manager_secret_version" "msal_client_secret" {
  secret      = google_secret_manager_secret.msal_client_secret.id
  secret_data = var.msal_client_secret
}

resource "google_secret_manager_secret_iam_binding" "msal_client_secret" {

  secret_id = google_secret_manager_secret.msal_client_secret.id
  role      = "roles/secretmanager.secretAccessor"
  members   = ["serviceAccount:${google_service_account.account.email}"]
}