# The MongoDB Atlas database user and the Atlas Search indexes the admin UI
# uses. The cluster itself, and the connection string stored as the
# mongodb_uri secret, are managed outside Terraform.
terraform {
  required_providers {
    mongodbatlas = {
      source = "mongodb/mongodbatlas"
    }
  }
}

variable "project_id" {
  type        = string
  description = "MongoDB Atlas project ID"
}

variable "cluster_name" {
  type        = string
  description = "Existing Atlas cluster the database lives on"
}

variable "database" {
  type        = string
  default     = "mrssanta"
  description = "Database name, also used as the user name"
}

resource "mongodbatlas_database_user" "this" {
  description        = "${var.database} user"
  username           = var.database
  project_id         = var.project_id
  auth_database_name = "admin"

  roles {
    role_name     = "${var.database}-readWrite"
    database_name = "admin"
  }

  scopes {
    name = var.cluster_name
    type = "CLUSTER"
  }
}

resource "mongodbatlas_search_index" "devices_index" {
  project_id      = var.project_id
  cluster_name    = var.cluster_name
  database        = var.database
  collection_name = "devices"
  name            = "devices_index"

  analyzer         = "lucene.standard"
  search_analyzer  = "lucene.standard"
  mappings_dynamic = false

  mappings_fields = <<-EOF
  {
    "_id": {
      "type": "objectId"
    },
    "identifier": {
      "type": "string",
      "analyzer": "lucene.keyword",
      "searchAnalyzer": "lucene.keyword",
      "store": true
    },
    "hostname": {
      "type": "string",
      "analyzer": "lucene.keyword",
      "searchAnalyzer": "lucene.keyword",
      "store": true
    }
  }
  EOF
}

resource "mongodbatlas_search_index" "rules_index" {
  project_id      = var.project_id
  cluster_name    = var.cluster_name
  database        = var.database
  collection_name = "rules"
  name            = "rules_index"

  analyzer         = "lucene.standard"
  search_analyzer  = "lucene.standard"
  mappings_dynamic = false

  mappings_fields = <<-EOF
  {
    "_id": {
      "type": "objectId"
    },
    "identifier": {
      "type": "string",
      "analyzer": "lucene.keyword",
      "searchAnalyzer": "lucene.keyword",
      "store": true
    }
  }
  EOF
}

resource "mongodbatlas_search_index" "events_index" {
  project_id      = var.project_id
  cluster_name    = var.cluster_name
  database        = var.database
  collection_name = "events"
  name            = "events_index"

  analyzer         = "lucene.standard"
  search_analyzer  = "lucene.standard"
  mappings_dynamic = false

  mappings_fields = <<-EOF
  {
    "_id": {
      "type": "objectId"
    },
    "file_sha256": {
      "type": "string",
      "analyzer": "lucene.keyword",
      "searchAnalyzer": "lucene.keyword",
      "store": true
    }
  }
  EOF
}