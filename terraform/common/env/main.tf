variable "type" {}
variable "root" {}
variable "dev_types" {
  type = list(string)
  default = [
    "dev",
    "local",
  ]
}

output "type" {
  value = var.type
}

output "root" {
  value = var.root
}

output "is_dev" {
  value = contains(var.dev_types, var.type)
}
