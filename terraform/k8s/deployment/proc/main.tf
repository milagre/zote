variable "env" {}
variable "name" {}
variable "namespace" {}
variable "profile" {}
variable "image" {}
variable "tag" {}
variable "cmd" {
  type    = list(string)
  default = null
}
variable "args" {
  type    = list(string)
  default = null
}
variable "conf" {
  type = object({
    configmaps = list(string)
    secrets    = list(string)
    values     = map(string)
  })
}
variable "files" {
  type = object({
    configmaps = map(string),
  })
  default = {
    configmaps = {},
  }
}

locals {
}

module "cloudconstants" {
  source = "../../../cloud/constants"

  env = var.env
}
