
variable "namespace" {}
variable "account_id" { type = string }
variable "api_token" { type = string }
variable "tunnel_name" { type = string }


resource "helm_release" "cloudflare-tunnel" {
  chart      = "cloudflare-tunnel-ingress-controller"
  name       = "cloudflare-tunnel"
  namespace  = var.namespace
  repository = "https://helm.strrl.dev/"

  set {
    name  = "cloudflare.apiToken"
    value = var.api_token
  }

  set {
    name  = "cloudflare.accountId"
    value = var.account_id
  }

  set {
    name  = "cloudflare.tunnelName"
    value = var.tunnel_name
  }
}

