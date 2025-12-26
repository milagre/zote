variable "env" {} # tflint-ignore: terraform_unused_declarations
variable "name" {}
variable "namespace" {} # tflint-ignore: terraform_unused_declarations
variable "database" {}
variable "vpc_id" {}
variable "project_id" {}
variable "ver" {}
variable "primary" {}
variable "replicas" {}

data "digitalocean_vpc" "vpc" {
  id = var.vpc_id
}

data "digitalocean_project" "project" {
  id = var.project_id
}

locals {
  name = "${data.digitalocean_vpc.vpc.name}-${var.name}"
}

resource "digitalocean_database_cluster" "cluster" {
  name                 = local.name
  engine               = "mysql"
  version              = var.ver
  size                 = var.primary.class
  region               = data.digitalocean_vpc.vpc.region
  node_count           = 1
  private_network_uuid = data.digitalocean_vpc.vpc.id
  project_id           = data.digitalocean_project.project.id

  lifecycle {
    prevent_destroy = true
  }
}

resource "digitalocean_database_firewall" "vpc_fw" {
  cluster_id = digitalocean_database_cluster.cluster.id

  rule {
    type  = "ip_addr"
    value = data.digitalocean_vpc.vpc.ip_range
  }
}

resource "digitalocean_database_mysql_config" "cfg" {
  cluster_id = digitalocean_database_cluster.cluster.id

  sql_mode = join(",", [
    "ANSI_QUOTES",
    "ERROR_FOR_DIVISION_BY_ZERO",
    "IGNORE_SPACE",
    "NO_ENGINE_SUBSTITUTION",
    "NO_ZERO_DATE",
    "NO_ZERO_IN_DATE",
    "ONLY_FULL_GROUP_BY",
    "PIPES_AS_CONCAT",
    "REAL_AS_FLOAT",
  ])
}

resource "digitalocean_database_db" "db" {
  cluster_id = digitalocean_database_cluster.cluster.id
  name       = var.database
}

resource "digitalocean_database_replica" "replicas" {
  count = var.replicas.num

  cluster_id = digitalocean_database_cluster.cluster.id
  name       = "${local.name}-replica-${count.index}"
  size       = var.replicas.class
  region     = data.digitalocean_vpc.vpc.region
}

output "username" {
  value     = digitalocean_database_cluster.cluster.user
  sensitive = false
}

output "hostname" {
  value     = digitalocean_database_cluster.cluster.private_host
  sensitive = false
}

output "port" {
  value     = digitalocean_database_cluster.cluster.port
  sensitive = false
}

output "password" {
  value     = digitalocean_database_cluster.cluster.password
  sensitive = true
}
