variable "env" {}
variable "name" {}
variable "namespace" {}
variable "container" {
  type = object({
    primary = object({
      profile = object({
        cpu = object({
          min = string
          max = string
        })
        mem = object({
          min = string
          max = string
        })
      })
    })
    replica = object({
      profile = object({
        cpu = object({
          min = string
          max = string
        })
        mem = object({
          min = string
          max = string
        })
      })
    })
  })
  default = null
}
variable "cloud" {
  type = object({
    digitalocean = optional(object({
      vpc_id     = string
      project_id = string
      version    = string
      primary = object({
        class = string
      })
      replicas = object({
        num   = number
        class = string
      })
    }))
  })
  default = null
}
variable "database" {
  type = string
}
variable "username" {
  type = string
}

locals {
  name = "mysql-${var.name}"
  cfg  = "${var.env.prefix}_${upper(var.namespace)}_${upper(var.name)}_MYSQL"

  dbmodule = coalesce(
    try(module.container[0], null),
    try(module.digitalocean[0], null),
  )
}

resource "kubernetes_config_map" "client" {
  metadata {
    name      = local.name
    namespace = var.namespace

    labels = {
      app = local.name
    }
  }

  data = {
    "${local.cfg}_HOST"     = local.dbmodule.hostname
    "${local.cfg}_PORT"     = local.dbmodule.port
    "${local.cfg}_USER"     = local.dbmodule.username
    "${local.cfg}_DATABASE" = var.database
  }
}

resource "kubernetes_secret" "client" {
  metadata {
    name      = local.name
    namespace = var.namespace

    labels = {
      app = local.name
    }
  }

  data = {
    "${local.cfg}_PASS" = local.dbmodule.password
  }
}

output "k8s" {
  value = {
    configmap = kubernetes_config_map.client.metadata[0].name
    secret    = kubernetes_secret.client.metadata[0].name
  }
}
