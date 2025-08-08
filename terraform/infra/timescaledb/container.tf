module "container" {
  count = var.container == null ? 0 : 1

  source = "./_container"

  env       = var.env
  name      = var.name
  namespace = var.namespace

  ver     = var.ver
  profile = var.container.profile

  user     = var.user
  database = var.database
}
