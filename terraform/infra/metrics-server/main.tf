variable "env" {} # tflint-ignore: terraform_unused_declarations
variable "namespace" {}

resource "helm_release" "metrics-server" {
  chart      = "metrics-server"
  name       = "metrics-server"
  namespace  = var.namespace
  repository = "https://kubernetes-sigs.github.io/metrics-server"
  version    = "3.11.0"
}
