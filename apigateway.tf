resource "local_file" "api_gateway_config" {
  content = templatefile("${path.module}/gcp/apigateway/template.yaml", {
    PREFLIGHT_URL = google_cloudfunctions2_function.preflight.url
    EVENTUPLOAD_URL = google_cloudfunctions2_function.eventupload.url
    RULEDOWNLOAD_URL = google_cloudfunctions2_function.ruledownload.url
    POSTFLIGHT_URL = google_cloudfunctions2_function.postflight.url
    XSRF_URL = google_cloudfunctions2_function.xsrf.url
    BLOCKABLES_URL = google_cloudfunctions2_function.blockables.url
    API_URL =  google_cloudfunctions2_function.api.url
    OAUTH_TENANT = var.oauth_tenant
    OAUTH_CLIENT_ID = var.oauth_client_id
  })
  filename = "${path.module}/gcp/apigateway/openapi.yaml"

  depends_on = [google_cloudfunctions2_function.preflight]
}

# resource "google_apikeys_key" "client" {
#   name         = "key"
#   display_name = "${var.service}-client-key"
# }

# resource "google_apikeys_key" "management" {
#   name         = "key"
#   display_name = "${var.service}-management-key"
# }

resource "google_api_gateway_api" "api_gw" {
  provider     = google-beta
  api_id       = var.service
  project      = var.project_id
  display_name = "${var.service}-api"

  labels = "${merge(var.labels, {
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.environment}"
    owner = "${var.owner}"
    team = "${var.team}"
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
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.environment}"
    owner = "${var.owner}"
    team = "${var.team}"
  })}"
}

resource "google_api_gateway_gateway" "gw" {
  provider = google-beta
  region   = "europe-west1"
  project  = var.project_id

  api_config   = google_api_gateway_api_config.api_cfg.id

  gateway_id   = "${var.service}-api"
  display_name = "${var.service}-api"
  
  labels = "${merge(var.labels, {
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.environment}"
    owner = "${var.owner}"
    team = "${var.team}"
  })}"

  depends_on   = [google_api_gateway_api_config.api_cfg]
}
