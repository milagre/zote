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

locals {
  name = "redis-${var.name}"
  cfg  = "${var.env.prefix}_REDIS_${join("_", split("-", upper(var.name)))}"

  targetmodule = coalesce(
    try(module.container[0], null),
  )
}

resource "kubernetes_config_map" "cfg" {
  metadata {
    name      = local.name
    namespace = var.namespace
  }

  data = {
    "${local.cfg}_HOST" = local.targetmodule.hostname
    "${local.cfg}_PORT" = local.targetmodule.port
  }
}

output "k8s" {
  value = {
    configmap = kubernetes_config_map.cfg.metadata[0].name
  }
}
