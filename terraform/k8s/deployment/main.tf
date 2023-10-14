variable "env" {}
variable "http" {
  type = object({
    port   = number
    health = string
  })
  default = null
}
variable "name" {}
variable "namespace" {}
variable "image" {}
variable "tag" {}
variable "ngrok" {
  type    = bool
  default = false
}
variable "public_domain" {}
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

module "http" {
  count = var.http != null ? 1 : 0

  source = "./http"

  name      = var.name
  namespace = var.namespace
  image     = var.image
  tag       = var.tag

  env = var.env

  setup = var.http
  ngrok = var.ngrok

  conf    = var.conf
  files   = var.files
  profile = module.profile
  cmd     = var.cmd
  args    = var.args

  internal = {
    public_hostname  = local.public_hostname
    private_hostname = local.private_hostname
  }
}

locals {
  public_hostname  = "${var.name}.${var.namespace}.${var.public_domain}"
  private_hostname = "${var.name}.${var.namespace}.svc.cluster.local"
}

output "public_hostname" {
  value = local.public_hostname
}

output "private_hostname" {
  value = local.private_hostname
}
