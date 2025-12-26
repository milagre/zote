terraform {
  required_version = ">= 1.12"
}

variable "env" {}

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
