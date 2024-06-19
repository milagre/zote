
variable "namespace" {}
variable "api_key" { type = string }
variable "auth_token" { type = string }
variable "domain" {}

resource "kubernetes_secret" "ngrok" {
  metadata {
    name      = "ingress-ngrok-kubernetes-ingress-controller-credentials"
    namespace = var.namespace
  }
  data = {
    API_KEY   = var.api_key
    AUTHTOKEN = var.auth_token
  }
}

resource "helm_release" "ngrok" {
  chart      = "kubernetes-ingress-controller"
  name       = "ingress-ngrok"
  namespace  = var.namespace
  repository = "https://ngrok.github.io/kubernetes-ingress-controller"
  version    = "0.12.0"

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

