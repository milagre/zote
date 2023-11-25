module "digitalocean" {
  count = var.cloud == null ? 0 : var.cloud.digitalocean == null ? 0 : 1

  source = "./_cloud/digitalocean"

  env       = var.env
  name      = local.name
  namespace = var.namespace

  database = var.database

  vpc_id     = var.cloud.digitalocean.vpc_id
  project_id = var.cloud.digitalocean.project_id
  vers       = var.cloud.digitalocean.version
  primary    = var.cloud.digitalocean.primary
  replicas   = var.cloud.digitalocean.replicas
}

