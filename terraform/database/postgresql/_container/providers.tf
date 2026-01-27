terraform {
  required_version = ">= 1.12"
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = ">= 2.23"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}
