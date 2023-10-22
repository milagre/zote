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
}

resource "random_password" "password" {
  length  = 64
  special = true

  min_special = 8
  min_numeric = 8
  min_lower   = 8
  min_upper   = 8

  override_special = "$%&*()-_=+[]{}<>:?"
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
    "${local.cfg}_HOST"     = "${local.name}.svc.cluster.local"
    "${local.cfg}_PORT"     = "3306"
    "${local.cfg}_DATABASE" = var.database
    "${local.cfg}_USER"     = var.username
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
    "${local.cfg}_PASS" = random_password.password.result
  }
}

output "k8s" {
  value = {
    configmap = kubernetes_config_map.client.metadata[0].name
    secret    = kubernetes_secret.client.metadata[0].name
  }
}
