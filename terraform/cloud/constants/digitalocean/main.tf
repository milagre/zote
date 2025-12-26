terraform {
  required_version = ">= 1.12"
}

variable "env" {} # tflint-ignore: terraform_unused_declarations

output "public_load_balancer_annotations" {
  value = {
    "service.beta.kubernetes.io/do-loadbalancer-tls-ports"       = "443",
    "service.beta.kubernetes.io/do-loadbalancer-tls-passthrough" = "true",
  }
}

output "private_load_balancer_annotations" {
  value = {
  }
}
