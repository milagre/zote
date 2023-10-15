variable "env" {}

module "digitalocean" {
  source = "./digitalocean"

  env = var.env
}


output "public_load_balancer_annotations" {
  value = var.env.is_local ? {} : merge(
    module.digitalocean.public_load_balancer_annotations,
    {},
  )
}

output "private_load_balancer_annotations" {
  value = var.env.is_local ? {} : merge(
    module.digitalocean.private_load_balancer_annotations,
    {},
  )
}
