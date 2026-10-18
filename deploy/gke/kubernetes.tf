data "google_container_cluster" "default" {
  name     = var.shared_service_k8s_cluster_name
  location = var.zone
}

data "google_client_config" "provider" {}

provider "kubernetes" {
  host                   = "https://${data.google_container_cluster.default.endpoint}"
  token                  = data.google_client_config.provider.access_token
  client_certificate     = base64decode(data.google_container_cluster.default.master_auth.0.client_certificate)
  client_key             = base64decode(data.google_container_cluster.default.master_auth.0.client_key)
  cluster_ca_certificate = base64decode(data.google_container_cluster.default.master_auth.0.cluster_ca_certificate)
}