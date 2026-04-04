output "namespace" {
  value       = var.namespace
  description = "Kubernetes namespace for this service stack."
}

output "api_public_hostnames" {
  value = module.api.public_hostnames
}

output "api_private_hostname" {
  value = module.api.private_hostname
}
