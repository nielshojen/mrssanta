resource "google_artifact_registry_repository" "images" {
  location      = var.region
  repository_id = "${var.service}-artifacts"
  description   = "Repository to hold container images for ${var.service}"
  format        = "DOCKER"
  cleanup_policy_dry_run = false
  cleanup_policies {
    id     = "delete-prerelease"
    action = "DELETE"
    condition {
      tag_state    = "TAGGED"
      tag_prefixes = ["v"]
      older_than   = "2592000s"
    }
  }
  cleanup_policies {
    id     = "keep-tagged-release"
    action = "KEEP"
    condition {
      tag_state             = "TAGGED"
      package_name_prefixes = ["mrssanta"]
    }
  }
  cleanup_policies {
    id     = "keep-minimum-versions"
    action = "KEEP"
    most_recent_versions {
      package_name_prefixes = ["mrssanta"] 
      keep_count            = 3
    }
  }

  labels = "${merge(var.labels, {
    env = "${var.environment}"
    app = "${var.service}"
    service = "${var.environment}"
    owner = "${var.owner}"
    team = "${var.team}"
  })}"
}

