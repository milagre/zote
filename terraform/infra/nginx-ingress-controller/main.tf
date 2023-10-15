
variable "namespace" {}
variable "env" {}

resource "helm_release" "nginx" {
  chart      = "ingress-nginx"
  name       = "ingress-nginx"
  namespace  = var.namespace
  repository = "https://kubernetes.github.io/ingress-nginx"
  version    = "4.7.2"

  dynamic "set" {
    for_each = var.env.is_local ? {
      "controller.service.type" = "NodePort"
    } : {}
    content {
      name  = set.key
      value = set.value
    }
  }
}
