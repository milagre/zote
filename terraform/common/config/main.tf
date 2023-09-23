
variable "env" {}

module "read" {
  source = "./read"

  env = var.env
}

output "data" {
  value = module.read.data
}
