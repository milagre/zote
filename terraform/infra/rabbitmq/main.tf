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
//variable "cloud" {
//  type = object({
//  })
//  default = null
//}

locals {
  name_amqp     = "amqp-${var.name}"
  name_rabbitmq = "rabbitmq-${var.name}"

  cfg_amqp     = "${var.env.prefix}_AMQP_${upper(var.name)}"
  cfg_rabbitmq = "${var.env.prefix}_RABBITMQ_${upper(var.name)}"

  targetmodule = coalesce(
    try(module.container[0], null),
    //try(module.digitalocean[0], null),
  )
}

resource "kubernetes_config_map_v1" "amqp" {
  metadata {
    name      = local.name_amqp
    namespace = var.namespace
  }

  data = {
    "${local.cfg_amqp}_HOST" = local.targetmodule.hostname
    "${local.cfg_amqp}_PORT" = local.targetmodule.port
  }
}

resource "kubernetes_config_map_v1" "rabbitmq" {
  metadata {
    name      = local.name_rabbitmq
    namespace = var.namespace
  }

  data = {
    "${local.cfg_rabbitmq}_HOST" = local.targetmodule.admin_hostname
    "${local.cfg_rabbitmq}_PORT" = local.targetmodule.admin_port
  }
}

resource "kubernetes_config_map_v1" "users" {
  for_each = toset([for user in var.setup.users : user.name])

  metadata {
    name      = "${local.name_amqp}-${each.key}"
    namespace = var.namespace
  }

  data = {
    "${local.cfg_amqp}_USER" = each.key
  }
}

resource "kubernetes_secret_v1" "passwords" {
  for_each = toset([for user in var.setup.users : user.name])

  metadata {
    name      = "${local.name_amqp}-${each.key}"
    namespace = var.namespace
  }

  data = {
    "${local.cfg_amqp}_PASS" = local.targetmodule.passwords[each.key]
  }
}

output "k8s" {
  value = {
    rabbitmq = {
      configmap = kubernetes_config_map_v1.rabbitmq.metadata[0].name
    }
    amqp = {
      configmap = kubernetes_config_map_v1.amqp.metadata[0].name
    }
    users = {
      for user in var.setup.users :
      user.name => {
        configmap = kubernetes_config_map_v1.users[user.name].metadata[0].name
        secret    = kubernetes_secret_v1.passwords[user.name].metadata[0].name
      }
    }
  }
}

output "api" {
  value = {
    host = local.targetmodule.admin_hostname
    port = local.targetmodule.admin_port
  }
}

output "amqp" {
  value = {
    host = local.targetmodule.hostname
    port = local.targetmodule.port
  }
}

output "users" {
  value = {
    for user in var.setup.users :
    user.name => local.targetmodule.passwords[user.name]
  }
  sensitive = true
}
