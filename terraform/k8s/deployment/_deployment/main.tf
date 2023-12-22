variable "env" {}
variable "name" {}
variable "namespace" {}
variable "type" {}
variable "image" {}
variable "tag" {}
variable "cmd" {}
variable "args" {}
variable "profile" {}
variable "conf" {
  type = object({
    values     = map(string)
    configmaps = list(string)
    secrets    = list(string)
  })
}
variable "files" {}
variable "http_liveness_probe" {
  type = object({
    path = string
    port = number
    freq = optional(number)
  })
  default = null
}


locals {
  timestamptag = replace(timestamp(), "/[-:TZ]/", "")
  tag          = var.tag == "latest" ? "latest-${local.timestamptag}" : var.tag
}

resource "kubernetes_deployment" "deploy" {
  metadata {
    name      = var.name
    namespace = var.namespace
    labels = {
      app     = var.name
      deploy  = var.type
      version = local.tag
    }
  }

  spec {
    replicas = var.profile.num.min

    selector {
      match_labels = {
        app    = var.name
        deploy = var.type
      }
    }

    template {
      metadata {
        name      = var.name
        namespace = var.namespace
        labels = {
          app     = var.name
          deploy  = var.type
          version = local.tag
        }
      }

      spec {
        container {
          name = var.name

          image = "${var.image}:${var.tag}"

          image_pull_policy = var.env.is_local ? "Never" : "IfNotPresent"

          command = var.cmd
          args    = var.args

          resources {
            limits = {
              cpu    = var.profile.cpu_cores.max
              memory = "${var.profile.mem_mb.max}M"
            }
            requests = {
              cpu    = var.profile.cpu_cores.min
              memory = "${var.profile.mem_mb.min}M"
            }
          }

          dynamic "liveness_probe" {
            for_each = toset([for x in [var.http_liveness_probe] : x if x != null])
            content {
              http_get {
                path = liveness_probe.value.path
                port = liveness_probe.value.port
              }

              initial_delay_seconds = 5
              period_seconds        = coalesce(liveness_probe.value.freq, 15)
            }
          }

          // Attach configmaps to environment
          dynamic "env_from" {
            for_each = coalesce(var.conf.configmaps, [])
            content {
              config_map_ref {
                name = env_from.value
              }
            }
          }

          // Attach secrets to environment
          dynamic "env_from" {
            for_each = coalesce(var.conf.secrets, [])
            content {
              secret_ref {
                name = env_from.value
              }
            }
          }

          // Attach arbitrary env values
          dynamic "env" {
            for_each = coalesce(var.conf.values, {})
            content {
              name  = env.key
              value = env.value
            }
          }

          // Mount configmaps needed for files in container
          dynamic "volume_mount" {
            for_each = coalesce(var.files.configmaps, {})
            content {
              name       = volume_mount.value
              mount_path = volume_mount.key
            }
          }
        }

        // Attach configmaps needed for files to spec
        dynamic "volume" {
          for_each = coalesce(var.files.configmaps, {})
          content {
            name = volume.value
            config_map {
              name = volume.value
            }
          }
        }

        # Pods attempt to spread out across nodes when possible
        affinity {
          pod_anti_affinity {
            preferred_during_scheduling_ignored_during_execution {
              pod_affinity_term {
                label_selector {
                  match_expressions {
                    key      = "app"
                    operator = "In"
                    values   = [var.name]
                  }
                  match_expressions {
                    key      = "version"
                    operator = "In"
                    values   = [var.tag]
                  }
                }
                topology_key = "kubernetes.io/hostname"
              }
              weight = 100
            }
          }
        }
      }
    }
  }
}
