variable "env" {
  description = "Output object from the common env module."
  type        = any
}

variable "config" {
  description = "Output object from the common config module."
  type        = any
}

variable "namespace" {
  description = "Kubernetes namespace name for workloads (created in the root module)."
  type        = string
}
