
variable "namespace" {}
variable "env" {}

resource "helm_release" "nginx" {
  chart      = "ingress-nginx"
  name       = "ingress-nginx"
  namespace  = var.namespace
  repository = "https://kubernetes.github.io/ingress-nginx"
  version    = "4.7.2"

  dynamic "set" {
    for_each = merge(
      var.env.is_local ? {
        "controller.service.type" = "NodePort"
      } : {},
      {
        "controller.replicaCount" = 2

        "controller.autoscaling.enabled"                           = true
        "controller.autoscaling.minReplicas"                       = 2
        "controller.autoscaling.targetCPUUtilizationPercentage"    = 75
        "controller.autoscaling.targetMemoryUtilizationPercentage" = 75

        "controller.resources.requests.cpu"    = "200m"
        "controller.resources.requests.memory" = "196Mi"
        "controller.resources.limits.cpu"      = "250m"
        "controller.resources.limits.memory"   = "256Mi"
      }
    )
    content {
      name  = set.key
      value = set.value
    }
  }

  values = [
    yamlencode({ controller = { affinity = local.affinity } })
  ]
}

locals {
  affinity = {
    podAntiAffinity = {
      preferredDuringSchedulingIgnoredDuringExecution = [
        {
          weight = 100
          podAffinityTerm = {
            labelSelector = {
              matchExpressions = [
                {
                  key      = "app.kubernetes.io/name"
                  operator = "In"
                  values   = ["ingress-nginx"]
                },
                {
                  key      = "app.kubernetes.io/instance"
                  operator = "In"
                  values   = ["ingress-nginx"]
                },
                {
                  key      = "app.kubernetes.io/component"
                  operator = "In"
                  values   = ["controller"]
                },
              ]
            }
            topologyKey = "kubernetes.io/hostname"
          }
        },
      ]
    }
  }
}
