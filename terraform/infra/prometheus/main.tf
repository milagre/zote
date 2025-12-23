variable "namespace" { type = string }
variable "helm_chart_version" {
  type    = string
  default = "80.6.0"
}

resource "helm_release" "prometheus" {
  name       = "kube-prometheus-stack"
  repository = "https://prometheus-community.github.io/helm-charts"
  chart      = "kube-prometheus-stack"
  version    = var.helm_chart_version
  namespace  = var.namespace

  values = [
    yamlencode({
      prometheus = {
        prometheusSpec = {
          podMonitorNamespaceSelector = {}
          podMonitorSelector = {
            matchLabels = {}
          }
        }
      }
    })
  ]
}
