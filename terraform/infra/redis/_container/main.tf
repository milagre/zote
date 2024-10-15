variable "env" {}
variable "namespace" {}
variable "name" {}
variable "ver" {}
variable "profile" {}
variable "shards" {}
variable "replicas" {}

locals {
  name = "redis-${var.name}"
}

module "profile" {
  source = "./../../../structs/profile"

  profile = var.profile
}

output "hostname" {
  value     = "${local.name}.${var.namespace}.svc.cluster.local"
  sensitive = false
}

output "port" {
  value     = 6379
  sensitive = false
}
