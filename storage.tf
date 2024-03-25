resource "google_storage_bucket" "source" {
  name     = "${var.service}-source"
  location = "EU"
  
  labels = "${merge(var.labels, {
    env = "prod"
    app = "${var.service}"
  })}"
}