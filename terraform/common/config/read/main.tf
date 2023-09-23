
variable "env" {}

locals {
  config_files = [
    file("${local.config_folder}/1.main.yaml"),
    file("${local.config_folder}/4.backend.yaml"),
    file("${local.config_folder}/5.info.yaml"),
  ]
  config_folder = "${var.env.root}/env/${var.env.type}/${var.env.type == "local" ? "" : "${var.env.name}/"}"
  config = merge([
    for f in local.config_files :
    yamldecode(f)
  ]...)
}

output "data" {
  value = local.config
}

