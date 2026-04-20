
variable "namespace" {
  type = string
}

resource "helm_release" "grafana" {
  name       = "grafana"
  namespace  = var.namespace
  chart      = "grafana"
  repository = "https://grafana-community.github.io/helm-charts"
  version    = "12.1.1"

  values = []
}
