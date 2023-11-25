
variable "env" {}

locals {
  config_folders = [
    "${var.env.root}/env/${var.env.type}",
    "${var.env.root}/env/${var.env.type}/${var.env.tier}",
    "${var.env.root}/env/${var.env.type}/${var.env.tier}/${var.env.name}",
  ]
  config_files = flatten(
    [
      for folder in local.config_folders :
      [
        for file in sort(fileset(folder, "[0-9].*.yaml")) :
        "${folder}/${file}"
      ]
    ]
  )
  config_data = [
    for file in local.config_files :
    yamldecode(
      templatefile(file, {
        env = var.env
      })
    )
  ]
}

module "deepmerge" {
  source  = "cloudposse/config/yaml//modules/deepmerge"
  version = "0.2.0"
  maps    = local.config_data
}

output "sources" {
  value = local.config_files
}

output "raw" {
  value = local.config_data
}

output "data" {
  value = module.deepmerge.merged
}

