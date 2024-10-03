data "google_project" "project" {
}

resource "google_kms_crypto_key" "crypto_key" {
  name     = "${var.service}-kms-key"
  key_ring = google_kms_key_ring.key_ring.id
  purpose  = "ENCRYPT_DECRYPT"
}

resource "google_kms_key_ring" "key_ring" {
  name     = "${var.service}-kms-key-ring"
  location = "europe"
}

resource "google_kms_crypto_key_iam_binding" "firestore_cmek_keyuser" {
  crypto_key_id = google_kms_crypto_key.crypto_key.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"

  members = [
    "serviceAccount:service-${data.google_project.project.number}@gcp-sa-firestore.iam.gserviceaccount.com",
  ]
}

resource "google_firestore_database" "database" {
  project     = var.project_id
  name        = var.service
  location_id = "eur3"

  type                              = "FIRESTORE_NATIVE"
  concurrency_mode                  = "OPTIMISTIC"
  app_engine_integration_mode       = "DISABLED"
  point_in_time_recovery_enablement = "POINT_IN_TIME_RECOVERY_ENABLED"
  delete_protection_state           = "DELETE_PROTECTION_DISABLED"
  deletion_policy                   = "DELETE"

  cmek_config {
    kms_key_name = google_kms_crypto_key.crypto_key.id
  }

  depends_on = [
    google_kms_crypto_key_iam_binding.firestore_cmek_keyuser
  ]
}

# resource "google_firestore_index" "sn_index" {
#   project     = var.project_id
#   database   = google_firestore_database.database.name
#   collection = "cert_mappings"

#   fields {
#     field_path = "Mapped"
#     order      = "ASCENDING"
#   }

#   fields {
#     field_path = "SerialNumber"
#     order      = "ASCENDING"
#   }
# }