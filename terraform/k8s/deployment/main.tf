variable "env" {}
variable "http" {
  type = object({
    port   = number
    health = string
  })
  default = null
}
variable "name" {}
variable "image" {}
variable "tag" {}
variable "conf" {
  type = object({
    configmaps = list(string)
    secrets    = list(string)
    values     = map(string)
  })
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

  name    = var.name
  image   = var.image
  tag     = var.tag
  env     = var.env
  setup   = var.http
  conf    = var.conf
  profile = module.profile
  cmd     = var.cmd
  args    = var.args
}
