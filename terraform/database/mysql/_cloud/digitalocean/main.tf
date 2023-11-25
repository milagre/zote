variable "env" {}
variable "name" {}
variable "namespace" {}
variable "database" {}
variable "vpc_id" {}
variable "project_id" {}
variable "vers" {}
variable "primary" {}
variable "replicas" {}

data "digitalocean_vpc" "vpc" {
  id = var.vpc_id
}

data "digitalocean_project" "project" {
  id = var.project_id
}

resource "digitalocean_database_cluster" "cluster" {
  name                 = "${data.digitalocean_vpc.vpc.name}-${var.name}"
  engine               = "mysql"
  version              = var.vers
  size                 = var.primary.class
  region               = data.digitalocean_vpc.vpc.region
  node_count           = 1
  private_network_uuid = data.digitalocean_vpc.vpc.id
  project_id           = data.digitalocean_project.project.id
}

resource "digitalocean_database_firewall" "vpc_fw" {
  cluster_id = digitalocean_database_cluster.cluster.id

  rule {
    type  = "ip_addr"
    value = data.digitalocean_vpc.vpc.ip_range
  }
}

resource "digitalocean_database_db" "db" {
  cluster_id = digitalocean_database_cluster.cluster.id
  name       = var.database
}

resource "digitalocean_database_replica" "replicas" {
  count = var.replicas.num

  cluster_id = digitalocean_database_cluster.cluster.id
  name       = "${var.name}-replica-${count.index}"
  size       = var.replicas.class
  region     = data.digitalocean_vpc.vpc.region
}

output "username" {
  value     = digitalocean_database_cluster.cluster.user
  sensitive = false
}

output "hostname" {
  value     = digitalocean_database_cluster.cluster.host
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
