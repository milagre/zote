
resource "helm_release" "influxdb2" {
  namespace  = var.namespace
  name       = local.name
  repository = "https://helm.influxdata.com"
  chart      = "influxdb2"
  version    = "2.1.2"

  timeout = 300

  set_sensitive {
    name  = "adminUser.password"
    value = local.pass
  }

  set_sensitive {
    name  = "adminUser.token"
    value = local.token
  }

  values = [
    yamlencode({
      nameOverride     = local.name
      fullnameOverride = local.name
      adminUser = {
        organization = local.org
        bucket       = local.bucket
        user         = local.user
      }
      image = {
        tag = "${var.ver}-alpine"
      }
      resources = {
        limits = {
          memory = "${module.profile.mem_mb.max}Mi"
          cpu    = module.profile.cpu_cores.max

        }
        requests = {
          memory = "${module.profile.mem_mb.min}Mi"
          cpu    = module.profile.cpu_cores.min
        }
      }
      extraEnvVars = [
        {
          name  = "INFLUXD_STORAGE_CACHE_MAX_MEMORY_SIZE"
          value = "${floor(module.profile.mem_mb.max * 0.6 * 1024 * 1024)}"
        }
      ]
      livenessProbe = {
        path                = "/health"
        scheme              = "HTTP"
        initialDelaySeconds = 0
        periodSeconds       = 10
        timeoutSeconds      = 1
        failureThreshold    = 3
      }
      readinessProbe = {
        path                = "/health"
        scheme              = "HTTP"
        initialDelaySeconds = 0
        periodSeconds       = 10
        timeoutSeconds      = 1
        successThreshold    = 1
        failureThreshold    = 3
      }
      service = {
        type = "ClusterIP"
      }
    })
  ]
}
