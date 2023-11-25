module "container" {
  count = var.container == null ? 0 : 1

  source = "./_container"

  env       = var.env
  name      = local.name
  namespace = var.namespace

  primary_profile = var.container.primary.profile
  replica_profile = var.container.replica.profile

  database = var.database
  username = var.username
}

