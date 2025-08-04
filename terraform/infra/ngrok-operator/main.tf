
variable "namespace" {}
variable "api_key" { type = string }
variable "auth_token" { type = string }
variable "domain" {}

resource "kubernetes_secret" "ngrok" {
  metadata {
    name      = "ngrok-operator-credentials"
    namespace = var.namespace
  }
  data = {
    API_KEY   = var.api_key
    AUTHTOKEN = var.auth_token
  }
}

resource "helm_release" "ngrok" {
  chart      = "ngrok-operator"
  name       = "ngrok-operator"
  namespace  = var.namespace
  repository = "https://ngrok.github.io/ngrok-operator"
  version    = "0.18.0"

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

