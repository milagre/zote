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
  type = object({
    enabled = bool
    domain  = string
  })
  default = null
}
variable "conf" {
  type = object({
    configmaps = optional(list(string))
    secrets    = optional(list(string))
    values     = optional(map(string))
  })
  default = {}
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
  profile = module.profile
  cmd     = var.cmd
  args    = var.args
}



