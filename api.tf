provider "google" {
  project = var.project_id
  region  = var.region
  zone    = var.zone
}

resource "local_file" "api_gateway_config" {
  content = templatefile("${path.module}/api/template.yaml", {
    PREFLIGHT_URL = google_cloudfunctions2_function.preflight.url
    EVENTUPLOAD_URL = google_cloudfunctions2_function.eventupload.url
    RULEDOWNLOAD_URL = google_cloudfunctions2_function.ruledownload.url
    POSTFLIGHT_URL = google_cloudfunctions2_function.postflight.url
    XSRF_URL = google_cloudfunctions2_function.xsrf.url
    BLOCKABLES_URL = google_cloudfunctions2_function.blockables.url
  })
  filename = "${path.module}/api/openapi.yaml"

  depends_on = [google_cloudfunctions2_function.preflight]
}

resource "google_api_gateway_api" "api_gw" {
  provider     = google-beta
  api_id       = var.service
  project      = var.project_id
  display_name = "${var.service}-api"

  labels = "${merge(var.labels, {
    env = "prod"
    app = "${var.service}"
  })}"
}

resource "google_api_gateway_api_config" "api_cfg" {
  provider             = google-beta
  api                  = google_api_gateway_api.api_gw.api_id
  api_config_id_prefix = "api"
  project              = var.project_id
  display_name         = "${var.service}-api"
  gateway_config {
    backend_config {
      google_service_account = google_service_account.account.email
    }
  }
  openapi_documents {
    document {
      path     = "openapi.yaml"
      contents = base64encode(local_file.api_gateway_config.content)
    }
  }
  lifecycle {
    create_before_destroy = true
  }

  depends_on = [local_file.api_gateway_config]
  
  labels = "${merge(var.labels, {
    env = "prod"
    app = "${var.service}"
  })}"
}

resource "google_api_gateway_gateway" "gw" {
  provider = google-beta
  region   = var.region
  project  = var.project_id

  api_config   = google_api_gateway_api_config.api_cfg.id

  gateway_id   = "${var.service}-api"
  display_name = "${var.service}-api"
  
  labels = "${merge(var.labels, {
    env = "prod"
    app = "${var.service}"
  })}"

  depends_on   = [google_api_gateway_api_config.api_cfg]
}
