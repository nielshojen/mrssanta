resource "google_storage_bucket" "state" {
  name          = "${var.service}-state"
  location      = "EU"
  force_destroy = false

  public_access_prevention = "enforced"
}