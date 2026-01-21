terraform {
  required_version = ">= 1.12"
}

variable "env" {}
variable "vars" {
  default = {}
  type    = map(string)
}

module "read" {
  source = "./read"

  env = var.env
}

output "sources" {
  value = module.read.sources
}

output "raw" {
  value = module.read.raw
}

output "data" {
  value = module.read.data
}

output "env_vars" {
  value = var.vars
}
