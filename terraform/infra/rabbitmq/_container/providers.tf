terraform {
  required_version = ">= 1.12"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.23"
    }
    external = {
      source  = "hashicorp/external"
      version = ">= 2.3"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}
