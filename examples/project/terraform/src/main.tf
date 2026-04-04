terraform {
  required_version = ">= 1.12"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.27"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
  }
}

variable "root" {
  type = string
}

variable "env_type" {
  type    = string
  default = "local"
}

variable "env_tier" {
  type    = string
  default = "dev"
}

variable "env_name" {
  type    = string
  default = ""
}

module "env" {
  source = "../../../../terraform/common/env"

  root   = var.root
  type   = var.env_type
  tier   = var.env_tier
  name   = var.env_name
  prefix = "DEMO"
}

module "config" {
  source = "../../../../terraform/common/config"

  env = module.env
  vars = {
    "DEMO_LOG_FORMAT"       = "json"
    "DEMO_LOG_LEVEL"        = "info"
    "DEMO_STATSD_ADDR"      = "127.0.0.1:8125"
    "DEMO_STATSD_TAG_STYLE" = "datadog"
  }
}

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "zote"
}

provider "helm" {
  kubernetes {
    config_path    = "~/.kube/config"
    config_context = "zote"
  }
}

resource "kubernetes_namespace_v1" "service" {
  metadata {
    name = "service"
  }
}

module "service" {
  source = "./service"

  env       = module.env
  config    = module.config
  namespace = kubernetes_namespace_v1.service.metadata[0].name

  depends_on = [kubernetes_namespace_v1.service]
}

output "namespace" {
  value = kubernetes_namespace_v1.service.metadata[0].name
}

output "api_public_hostnames" {
  value = module.service.api_public_hostnames
}

output "api_private_hostname" {
  value = module.service.api_private_hostname
}
