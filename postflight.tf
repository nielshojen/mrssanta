data "archive_file" "postflight" {
  type        = "zip"
  output_path = "${path.module}/functions/postflight.zip"
  source_dir  = "${path.module}/functions/postflight"
}

resource "google_storage_bucket_object" "postflight" {
  name   = "postflight.zip"
  bucket = google_storage_bucket.source.name
  source = "${path.module}/functions/postflight.zip"
}

resource "google_cloudfunctions2_function" "postflight" {
  name = "${var.service}-postflight"
  location = var.region
  description = var.service

  build_config {
    runtime = "python312"
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
      GCP_PROJECT = var.project_id
      DB_PREFIX = var.service
    }
    all_traffic_on_latest_revision = true
    service_account_email = google_service_account.account.email
  }
  
  labels = "${merge(var.labels, {
    env = "prod"
    app = "${var.service}"
  })}"

  depends_on = [ google_storage_bucket_object.postflight ]
}
