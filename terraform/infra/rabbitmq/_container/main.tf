variable "env" {}
variable "namespace" {}
variable "name" {}
variable "ver" {}
variable "profile" {}
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
  name = "rabbitmq-${var.name}"
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
  value     = 5672
  sensitive = false
}

output "passwords" {
  value = {
    for user in var.setup.users :
    user.name => random_password.passwords[user.name].result
  }
  sensitive = true
}

output "admin_hostname" {
  value     = "${local.name}.${var.namespace}.svc.cluster.local"
  sensitive = false
}

output "admin_port" {
  value     = 15672
  sensitive = false
}

output "admin_username" {
  value     = ""
  sensitive = false
}

output "admin_password" {
  value     = ""
  sensitive = true
}
