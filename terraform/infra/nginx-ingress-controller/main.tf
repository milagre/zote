
variable "namespace" {}
variable "minikube" {
  type    = bool
  default = false
}

resource "helm_release" "nginx" {
  chart      = "ingress-nginx"
  name       = "ingress-nginx"
  namespace  = var.namespace
  repository = "https://kubernetes.github.io/ingress-nginx"
  version    = "4.7.2"

  dynamic "set" {
    for_each = var.minikube ? {
      "controller.service.type" = "NodePort"
    } : {}
    content {
      name  = set.key
      value = set.value
    }
  }
}
