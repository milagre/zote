variable "env" {}
variable "namespace" {}
variable "name" {}
variable "ver" {}
variable "profile" {}

variable "org" {
  default = "influxdb"
}
variable "bucket" {
  default = "influxdb"
}
variable "user" {
  default = "admin"
}

locals {
  name = "influxdb-${var.name}"
}

module "profile" {
  source = "./../../../structs/profile"

  profile = var.profile
}

resource "random_password" "password" {
  length  = 64
  special = false

  min_numeric = 8
  min_lower   = 8
  min_upper   = 8

  override_special = "$%&*()-_=+[]{}<>:?"
}

resource "random_password" "token" {
  length  = 64
  special = false

  min_numeric = 8
  min_lower   = 8
  min_upper   = 8

  override_special = "$%&*()-_=+[]{}<>:?"
}

locals {
  user   = var.user
  org    = var.org
  bucket = var.org
  pass   = random_password.password.result
  token  = random_password.token.result
}

output "scheme" {
  value = "http"
}

output "host" {
  value = local.name
}

output "port" {
  value = 80
}

output "org" {
  value = local.org
}

output "bucket" {
  value = local.bucket
}

output "user" {
  value = local.user
}

output "pass" {
  value     = local.pass
  sensitive = true
}

output "token" {
  value     = local.token
  sensitive = true
}
