data "archive_file" "preflight" {
  type        = "zip"
  output_path = "${path.module}/gcp/functions/preflight.zip"
  source_dir  = "${path.module}/gcp/functions/preflight"
}

resource "google_storage_bucket_object" "preflight" {
  name   = "preflight.zip"
  bucket = google_storage_bucket.source.name
  source = "${path.module}/gcp/functions/preflight.zip"
}

resource "google_cloudfunctions2_function" "preflight" {
  name = "${var.service}-preflight"
  location = var.region
  description = var.service

  build_config {
    runtime = "go121"
    entry_point = "preflight"
    source {
      storage_source {
        bucket = google_storage_bucket.source.name
        object = google_storage_bucket_object.preflight.name
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

  depends_on = [ google_storage_bucket_object.preflight ]
}
