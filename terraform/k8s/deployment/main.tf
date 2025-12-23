variable "env" {}
variable "name" {}
variable "namespace" {}
variable "image" {}
variable "tag" {}
variable "public_domain" {
  default = null
}
variable "veneers" {
  type    = list(string)
  default = []
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

// Process types
variable "http" {
  type = object({
    port   = number
    health = string
    freq   = optional(number)
  })
  default = null
}
variable "prometheus_monitored" {
  type    = bool
  default = false
}
variable "socket" {
  default = null
}


module "profile" {
  source = "./../../structs/profile"

  profile = var.profile
}

locals {
  type = (
    var.http != null ? "http" :
    var.socket != null ? "socket" :
    "proc"
  )
}

module "http" {
  count = local.type == "http" ? 1 : 0

  source = "./http"
  env    = var.env

  namespace = var.namespace
  name      = var.name

  image   = var.image
  tag     = var.tag
  cmd     = var.cmd
  args    = var.args
  profile = module.profile

  conf  = var.conf
  files = var.files

  setup = var.http

  internal = {
    public_hostname  = local.public_hostname
    private_hostname = local.private_hostname
    veneer_hostnames = var.veneers
  }

  prometheus_monitored = var.prometheus_monitored
}

module "proc" {
  count = local.type == "proc" ? 1 : 0

  source = "./proc"
  env    = var.env

  name      = var.name
  namespace = var.namespace

  image   = var.image
  tag     = var.tag
  cmd     = var.cmd
  args    = var.args
  profile = module.profile

  conf  = var.conf
  files = var.files

  prometheus_monitored = var.prometheus_monitored
}

locals {
  public_hostname  = var.public_domain != null ? "${var.name}.${var.namespace}.${var.public_domain}" : null
  private_hostname = "${var.name}.${var.namespace}.svc.cluster.local"
}

output "public_hostname" {
  value = local.public_hostname
}

output "private_hostname" {
  value = local.private_hostname
}
