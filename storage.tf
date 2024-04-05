resource "google_storage_bucket" "source" {
  name     = "${var.service}-source"
  location = "EU"
  
  labels = {
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.environment}"
    owner = "${var.owner}"
    team = "${var.team}"
    version = replace(var.service_version, ".", "-"),
  }
}