data "archive_file" "eventupload" {
  type        = "zip"
  output_path = "${path.module}/functions/eventupload.zip"
  source_dir  = "${path.module}/functions/eventupload"
}

resource "google_storage_bucket_object" "eventupload" {
  name   = "eventupload.zip"
  bucket = google_storage_bucket.source.name
  source = "${path.module}/functions/eventupload.zip"
}

resource "google_cloudfunctions2_function" "eventupload" {
  name = "${var.service}-eventupload"
  location = var.region
  description = var.service

  build_config {
    runtime = "python312"
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

  depends_on = [ google_storage_bucket_object.eventupload ]
}
