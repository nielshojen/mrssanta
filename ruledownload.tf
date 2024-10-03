data "archive_file" "ruledownload" {
  type        = "zip"
  output_path = "${path.module}/gcp/functions/ruledownload.zip"
  source_dir  = "${path.module}/gcp/functions/ruledownload"
}

resource "google_storage_bucket_object" "ruledownload" {
  name   = "ruledownload.zip"
  bucket = google_storage_bucket.source.name
  source = "${path.module}/gcp/functions/ruledownload.zip"
}

resource "google_cloudfunctions2_function" "ruledownload" {
  name = "${var.service}-ruledownload"
  location = var.region
  description = var.service

  build_config {
    runtime = "go121"
    entry_point = "ruledownload"
    source {
      storage_source {
        bucket = google_storage_bucket.source.name
        object = google_storage_bucket_object.ruledownload.name
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

  depends_on = [ google_storage_bucket_object.ruledownload ]
}
