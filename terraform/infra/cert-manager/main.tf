
variable "env" {}
variable "acme_email" {}
variable "namespace" {}

resource "helm_release" "cert-manager" {
  chart      = "cert-manager"
  name       = "cert-manager"
  namespace  = var.namespace
  repository = "https://charts.jetstack.io"
  version    = "1.13.1"

  set {
    name  = "installCRDs"
    value = "true"
  }
}

resource "kubernetes_manifest" "clusterissuer" {
  manifest = {
    apiVersion = "cert-manager.io/v1"
    kind       = "ClusterIssuer"

    metadata = {
      name = "letsencrypt-http01"
    }

    spec = {
      acme = {
        email  = var.acme_email
        server = "https://acme-v02.api.letsencrypt.org/directory"
        privateKeySecretRef = {
          name = "letsencrypt-http01"
        }
        solvers = [
          {
            http01 = {
              ingress = {
                class = "nginx"
              }
            }
          }
        ]
      }
    }
  }

  depends_on = [
    helm_release.cert-manager
  ]
}
