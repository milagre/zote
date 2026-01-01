variable "env" {}
variable "namespace" {}
variable "name" {}
variable "container" {
  type = object({
    profile = object({
      cpu = object({
        min = string
        max = string
      })
      mem = object({
        min = string
        max = string
      })
      num = object({
        min = number
        max = number
      })
    })
  })
  default = null
}
variable "user" {
  type    = string
  default = "timescaledb"
}
variable "database" {
  type    = string
  default = "timescaledb"
}
//variable "cloud" {
//  type = object({
//  })
//  default = null
//}

locals {
  name = "timescaledb-${var.name}"
  cfg  = "${var.env.prefix}_TIMESCALEDB_${upper(var.name)}"

  targetmodule = coalesce(
    try(module.container[0], null),
    //try(module.digitalocean[0], null),
  )
}

resource "kubernetes_config_map_v1" "client" {
  metadata {
    name      = local.name
    namespace = var.namespace

    labels = {
      app = local.name
    }
  }

  data = {
    "${local.cfg}_SCHEME"   = local.targetmodule.scheme
    "${local.cfg}_HOST"     = local.targetmodule.host
    "${local.cfg}_PORT"     = local.targetmodule.port
    "${local.cfg}_USER"     = local.targetmodule.user
    "${local.cfg}_DATABASE" = local.targetmodule.database
  }
}

resource "kubernetes_secret_v1" "client" {
  metadata {
    name      = local.name
    namespace = var.namespace

    labels = {
      app = local.name
    }
  }

  data = {
    "${local.cfg}_PASSWORD" = local.targetmodule.pass
  }
}

output "k8s" {
  value = {
    configmap = kubernetes_config_map.client.metadata[0].name
    secret    = kubernetes_secret.client.metadata[0].name
  }
}
