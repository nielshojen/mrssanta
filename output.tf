output "api-url" {
  value = google_api_gateway_gateway.gw.default_hostname
}

# output "gui-url" {
#   value = google_cloud_run_service.gui.status[0].url
# }
