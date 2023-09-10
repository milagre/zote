
variable "namespace" {}
variable "api_key" { type = string }
variable "auth_token" { type = string }
variable "domain" {}

resource "kubernetes_secret" "ngrok" {
  metadata {
    name      = "ngrok-ingress-controller-credentials"
    namespace = var.namespace
  }
  data = {
    API_KEY : var.api_key
    AUTHTOKEN : var.auth_token
  }
}

resource "helm_release" "ngrok" {
  chart      = "ngrok-ingress-controller"
  name       = "ngrok-ingress-controller"
  namespace  = var.namespace
  repository = "https://ngrok.github.io/kubernetes-ingress-controller"

  set {
    name  = "podSecurityPolicy.enabled"
    value = true
  }

  set {
    name  = "server.persistentVolume.enabled"
    value = false
  }

  depends_on = [kubernetes_secret.ngrok]
}

