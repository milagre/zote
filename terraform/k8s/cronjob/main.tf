variable "env" {}
variable "name" {}
variable "namespace" {}
variable "image" {}
variable "tag" {}
variable "schedule" {}
variable "timezone" {
  default = "Etc/UTC"
}
variable "conf" {
  type = object({
    configmaps = optional(list(string))
    secrets    = optional(list(string))
    values     = optional(map(string))
  })
  default = {}
}
variable "files" {
  type = object({
    configmaps = map(string),
  })
  default = {
    configmaps = {},
  }
}
variable "profile" {}
variable "cmd" {
  type    = list(string)
  default = null
}
variable "args" {
  type    = list(string)
  default = null
}

module "profile" {
  source = "./../../structs/profile"

  profile = var.profile
}

locals {
}

