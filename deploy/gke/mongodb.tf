provider "mongodbatlas" {
  public_key  = var.mongodb_public_key
  private_key = var.mongodb_private_key
}

module "atlas" {
  source = "../modules/atlas"

  project_id   = var.mongodb_project_id
  cluster_name = "PRODUCTION"
  database     = var.service
}

moved {
  from = mongodbatlas_database_user.mrssanta
  to   = module.atlas.mongodbatlas_database_user.this
}

moved {
  from = mongodbatlas_search_index.devices_index
  to   = module.atlas.mongodbatlas_search_index.devices_index
}

moved {
  from = mongodbatlas_search_index.rules_index
  to   = module.atlas.mongodbatlas_search_index.rules_index
}

moved {
  from = mongodbatlas_search_index.events_index
  to   = module.atlas.mongodbatlas_search_index.events_index
}
