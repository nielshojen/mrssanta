# Function Source
resource "google_storage_bucket" "source" {
  name     = "${var.service}-source"
  location = "EU"
  
  labels = "${merge(var.labels, {
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.environment}"
    owner = "${var.owner}"
    team = "${var.team}"
  })}"
}

# Serve Static Files
resource "google_storage_bucket" "static" {
  name     = "${var.service}-static"
  location = "EU"
  uniform_bucket_level_access = true
  
  labels = "${merge(var.labels, {
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.environment}"
    owner = "${var.owner}"
    team = "${var.team}"
  })}"
}

resource "google_storage_bucket_iam_binding" "public_access" {
  bucket = google_storage_bucket.static.name
  role   = "roles/storage.objectViewer"

  members = [
    "allUsers"
  ]
}

resource "google_storage_bucket_object" "png" {
  for_each = fileset(path.module, "gcp/functions/blockables/static/*.png")
  name   = trimprefix(each.value, "gcp/functions/blockables/static/")
  bucket = google_storage_bucket.static.name
  source = "${each.value}"
  content_type = "image/png"
}

resource "google_storage_bucket_object" "ico" {
  for_each = fileset(path.module, "gcp/functions/blockables/static/*.ico")
  name   = trimprefix(each.value, "gcp/functions/blockables/static/")
  bucket = google_storage_bucket.static.name
  source = "${each.value}"
  content_type = "image/x-icon"
}

resource "google_storage_bucket_object" "webmanifest" {
  for_each = fileset(path.module, "gcp/functions/blockables/static/*.webmanifest")
  name   = trimprefix(each.value, "gcp/functions/blockables/static/")
  bucket = google_storage_bucket.static.name
  source = "${each.value}"
  content_type = "application/manifest+json"
}