variable "project_id" {
  type        = string
  description = "Google Cloud Project ID"
}

variable "region" {
  type        = string
  description = "Google Cloud Region"
}

variable "zone" {
  type        = string
  description = "Google Cloud Region"
}

variable "domain" {
  type        = string
}

variable "labels" {
  type = map(string)
  description = "The labels to apply to the resources"
}

variable "service" {
  type        = string
  default     = "mrssanta"
  description = "The name of the service"
}

variable "service_version" {
  type        = string
  description = "Service version"
  default = "0.0.1"
}