variable "type" {}
variable "tier" {}
variable "name" {}
variable "minikube" { type = bool }
variable "root" {}
variable "dev_types" {
  type = list(string)
  default = [
    "dev",
    "local",
  ]
}

locals {
  is_dev   = var.tier == "dev"
  is_local = var.type == "local"

  id = "${var.tier}-${var.name}"
}

output "type" {
  value = var.type
}

output "tier" {
  value = var.tier
}

output "name" {
  value = var.name
}

output "id" {
  value = local.id
}

output "root" {
  value = var.root
}

output "is_dev" {
  value = local.is_dev
}

output "is_local" {
  value = local.is_local
}

output "lb_type" {
  value = var.minikube ? "NodePort" : "LoadBalancer"
}
