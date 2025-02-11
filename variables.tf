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

variable "service" {
  type        = string
  default     = "mrssanta"
  description = "The name of the service"
}

variable "environment" {
  type        = string
  description = "Release Environment (eg. dev or prod)"
}

variable "owner" {
  type        = string
  description = "The application owner"
}

variable "team" {
  type        = string
  description = "The application owner team"
}

variable "service_version" {
  type        = string
  description = "Service version"
  default = "0.1"
}

variable "client_id" {
  type        = string
  description = "MSAL Client ID"
}

variable "tenant_id" {
  type        = string
  description = "MSAL Tenant ID"
}

variable "virustotal_api_key" {
  type        = string
  description = "VirusTotal API Key"
}

variable "labels" {
  type = map(string)
  description = "The labels to apply to the resources"
}