data "archive_file" "blockables" {
  type        = "zip"
  output_path = "${path.module}/gcp/functions/blockables.zip"
  source_dir  = "${path.module}/gcp/functions/blockables"
}

resource "google_storage_bucket_object" "blockables" {
  name   = "blockables.zip"
  bucket = google_storage_bucket.source.name
  source = "${path.module}/gcp/functions/blockables.zip"
}

resource "google_cloudfunctions2_function" "blockables" {
  name = "${var.service}-blockables"
  location = var.region
  description = var.service

  build_config {
    runtime = "python312"
    entry_point = "blockables"
    source {
      storage_source {
        bucket = google_storage_bucket.source.name
        object = google_storage_bucket_object.blockables.name
      }
    }
  }

  service_config {
    max_instance_count  = 10
    available_memory    = "256M"
    timeout_seconds     = 60
    environment_variables = {
      ORGANIZATION  = var.organization
      VOTE_THRESHOLD = var.vote_threshold
      LOG_EXECUTION_ID = "true"
      GCP_PROJECT = var.project_id
      MONGO_DB = var.service
      ENVIRONMENT = var.environment
      OWNER = var.owner
      TEAM = var.team
      VERSION = var.service_version
    }
    secret_environment_variables {
      key        = "MONGO_URI"
      project_id = var.project_id
      secret     = google_secret_manager_secret.mongodb_uri.secret_id
      version    = "latest"
    }
    secret_environment_variables {
      key        = "API_KEY"
      project_id = var.project_id
      secret     = google_secret_manager_secret.api_key.secret_id
      version    = "latest"
    }
    secret_environment_variables {
      key        = "VT_API_KEY"
      project_id = var.project_id
      secret     = google_secret_manager_secret.virustotal_api_key.secret_id
      version    = "latest"
    }
    vpc_connector = data.google_vpc_access_connector.connector.id
    vpc_connector_egress_settings = "PRIVATE_RANGES_ONLY"
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

  depends_on = [ google_storage_bucket_object.blockables ]
}
