variable "type" {}
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
  is_dev = contains(var.dev_types, var.type)
}

output "type" {
  value = var.type
}

output "root" {
  value = var.root
}

output "is_dev" {
  value = local.is_dev
}

output "lb_type" {
  value = var.minikube ? "NodePort" : "External"
}
