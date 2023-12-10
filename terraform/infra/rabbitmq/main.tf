variable "env" {}
variable "namespace" {}
variable "name" {}
variable "ver" {}
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
variable "cloud" {
  type = object({
  })
  default = null
}
variable "setup" {
  type = object({
    vhosts = list(object({
      name  = string
      users = list(string)
    }))
    users = list(object({
      name = string
      tags = list(string)
    }))
  })
}

locals {
  name       = "rabbitmq-${var.name}"
  client_cfg = "RABBITMQ_${upper(var.name)}"
  admin_cfg  = "RABBITMQ_ADMIN_${upper(var.name)}"

  targetmodule = coalesce(
    try(module.container[0], null),
    //try(module.digitalocean[0], null),
  )
}

resource "kubernetes_config_map" "common" {
  metadata {
    name      = local.name
    namespace = var.namespace
  }

  data = {
    "${local.client_cfg}_HOST" = local.targetmodule.hostname
    "${local.client_cfg}_PORT" = local.targetmodule.port
  }
}

resource "kubernetes_config_map" "users" {
  for_each = toset([for user in var.setup.users : user.name])

  metadata {
    name      = "${local.name}-${each.key}"
    namespace = var.namespace
  }

  data = {
    "${local.client_cfg}_USER" = each.key
  }
}

resource "kubernetes_secret" "passwords" {
  for_each = toset([for user in var.setup.users : user.name])

  metadata {
    name      = "${local.name}-${each.key}"
    namespace = var.namespace
  }

  data = {
    "${local.client_cfg}_PASS" = local.targetmodule.passwords[each.key]
  }
}

output "k8s" {
  value = {
    configmap = kubernetes_config_map.common.metadata[0].name
    users = {
      for user in var.setup.users :
      "${user.name}" => {
        configmap = kubernetes_config_map.users[user.name].metadata[0].name
        secret    = kubernetes_secret.passwords[user.name].metadata[0].name
      }
    }
  }
}
