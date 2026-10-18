# Serve Static Files
resource "google_storage_bucket" "static" {
  name     = "${var.service}-static"
  location = "EU"
  uniform_bucket_level_access = true
  
  labels = "${merge(var.labels, {
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.service}"
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
  for_each = fileset(path.module, "static/*.png")
  name   = trimprefix(each.value, "static/")
  bucket = google_storage_bucket.static.name
  source = "${each.value}"
  source_md5hash = filemd5(each.value) # re-upload when the file changes
  content_type = "image/png"
}

resource "google_storage_bucket_object" "gif" {
  for_each = fileset(path.module, "static/*.gif")
  name   = trimprefix(each.value, "static/")
  bucket = google_storage_bucket.static.name
  source = "${each.value}"
  source_md5hash = filemd5(each.value) # re-upload when the file changes
  content_type = "image/gif"
}

resource "google_storage_bucket_object" "ico" {
  for_each = fileset(path.module, "static/*.ico")
  name   = trimprefix(each.value, "static/")
  bucket = google_storage_bucket.static.name
  source = "${each.value}"
  source_md5hash = filemd5(each.value) # re-upload when the file changes
  content_type = "image/x-icon"
}

resource "google_storage_bucket_object" "webmanifest" {
  for_each = fileset(path.module, "static/*.webmanifest")
  name   = trimprefix(each.value, "static/")
  bucket = google_storage_bucket.static.name
  source = "${each.value}"
  source_md5hash = filemd5(each.value) # re-upload when the file changes
  content_type = "application/manifest+json"
}