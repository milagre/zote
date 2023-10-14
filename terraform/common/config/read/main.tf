
variable "env" {}

locals {
  config_files  = sort(fileset(local.config_folder, "[0-9].*.yaml"))
  config_folder = "${var.env.root}/env/${var.env.type}/${var.env.type == "local" ? "" : "${var.env.tier}/"}"
  config_data = [
    for f in local.config_files :
    yamldecode(file("${local.config_folder}/${f}"))
  ]
}

module "deepmerge" {
  source = "github.com/cloudposse/terraform-yaml-config/modules/deepmerge"
  maps   = local.config_data
}

output "data" {
  value = module.deepmerge.merged
}

