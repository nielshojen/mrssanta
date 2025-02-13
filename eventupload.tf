data "archive_file" "eventupload" {
  type        = "zip"
  output_path = "${path.module}/gcp/functions/eventupload.zip"
  source_dir  = "${path.module}/gcp/functions/eventupload"
}

resource "google_storage_bucket_object" "eventupload" {
  name   = "eventupload.zip"
  bucket = google_storage_bucket.source.name
  source = "${path.module}/gcp/functions/eventupload.zip"
}

resource "google_cloudfunctions2_function" "eventupload" {
  name = "${var.service}-eventupload"
  location = var.region
  description = var.service

  build_config {
    runtime = "go122"
    entry_point = "eventupload"
    source {
      storage_source {
        bucket = google_storage_bucket.source.name
        object = google_storage_bucket_object.eventupload.name
      }
    }
  }

  service_config {
    max_instance_count  = 10
    available_memory    = "256M"
    timeout_seconds     = 60
    environment_variables = {
      LOG_EXECUTION_ID = "true"
      GCP_PROJECT = var.project_id
      FIRESTORE_DATABASE = google_firestore_database.database.name
      DB_PREFIX = var.service
      ENVIRONMENT = var.environment
      OWNER = var.owner
      TEAM = var.team
      VERSION = var.service_version
    }
    secret_environment_variables {
      key        = "API_KEY"
      project_id = var.project_id
      secret     = google_secret_manager_secret.api_key.secret_id
      version    = "latest"
    }
    all_traffic_on_latest_revision = true
    service_account_email = google_service_account.account.email
  }
  
  labels = "${merge(var.labels, {
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.environment}"
    owner = "${var.owner}"
    team = "${var.team}"
    version = replace(var.service_version, ".", "-"),
  })}"

  depends_on = [ google_storage_bucket_object.eventupload ]
}
