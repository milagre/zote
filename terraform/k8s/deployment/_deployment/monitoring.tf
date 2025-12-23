locals {
  has_metrics_port = length([for p in var.ports : p.name if p.name == "metrics"]) > 0
}

resource "kubernetes_manifest" "podmonitor" {
  count = local.has_metrics_port ? 1 : 0

  manifest = {
    apiVersion = "monitoring.coreos.com/v1"
    kind       = "PodMonitor"
    metadata = {
      name      = var.name
      namespace = var.namespace
      labels = {
        release = "prometheus"
        app     = var.name
      }
    }
    spec = {
      selector = {
        matchLabels = local.labels
      }
      namespaceSelector = {
        matchNames = [var.namespace]
      }
      podMetricsEndpoints = [
        {
          port = "metrics"
          path = "/metrics"
        }
      ]
    }
  }
}
