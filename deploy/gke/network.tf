data "google_compute_address" "ingress_ip" {
  name =  "${var.shared_service_name}-kubernetes-ingress-external-ip"
}
