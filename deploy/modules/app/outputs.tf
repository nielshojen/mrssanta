output "namespace" {
  value = kubernetes_namespace_v1.this.metadata[0].name
}

output "sync_base_url" {
  value       = "https://${var.host}/santa/"
  description = "Santa's SyncBaseURL"
}
