data "archive_file" "postflight" {
  type        = "zip"
  output_path = "${path.module}/gcp/functions/postflight.zip"
  source_dir  = "${path.module}/gcp/functions/postflight"
}

resource "google_storage_bucket_object" "postflight" {
  name   = "postflight.zip"
  bucket = google_storage_bucket.source.name
  source = "${path.module}/gcp/functions/postflight.zip"
}

resource "google_cloudfunctions2_function" "postflight" {
  name = "${var.service}-postflight"
  location = var.region
  description = var.service

  build_config {
    runtime = "go122"
    entry_point = "postflight"
    source {
      storage_source {
        bucket = google_storage_bucket.source.name
        object = google_storage_bucket_object.postflight.name
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
      DB_PREFIX = var.service
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

  depends_on = [ google_storage_bucket_object.postflight ]
}
