terraform {
  required_version = ">= 1.12"
}

variable "type" {}
variable "tier" {}
variable "name" {}
variable "root" {}
variable "prefix" {}

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
  value = local.is_local ? "NodePort" : "LoadBalancer"
}

output "prefix" {
  value = var.prefix
}
