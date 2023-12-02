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
variable "setup" {
  type = object({
    port   = number
    health = string
    freq   = optional(number)
  })
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
variable "ngrok" {
  type    = bool
  default = false
}
variable "internal" {
  type = object({
    public_hostname  = string
    private_hostname = string
    veneer_hostnames = list(string)
  })
}

locals {
  timestamptag = replace(timestamp(), "/[-:TZ]/", "")
  tag          = var.tag == "latest" ? "latest-${local.timestamptag}" : var.tag
}

module "cloudconstants" {
  source = "../../../cloud/constants"

  env = var.env
}
