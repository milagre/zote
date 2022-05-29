variable "env" {}
variable "name" {}
variable "profile" {}
variable "image" {}
variable "tag" {}
variable "cmd" {
  type    = list(string)
  default = null
}
variable "args" {
  type    = list(string)
  default = null
}
variable "setup" {
  type = object({
    port   = number
    health = string
  })
}
variable "conf" {
  type = object({
    configmaps = list(string)
    secrets    = list(string)
    values     = map(string)
  })
}

locals {
  timestamptag = replace(timestamp(), "/[-:TZ]/", "")
}

resource "kubernetes_deployment" "deploy" {
  metadata {
    name      = var.name
    namespace = var.env.namespace
    labels = {
      app     = var.name
      deploy  = "http"
      version = var.tag == "latest" ? "latest-${local.timestamptag}" : var.tag
    }
  }

  spec {
    replicas = var.profile.num.min

    selector {
      match_labels = {
        app    = var.name
        deploy = "http"
      }
    }

    template {
      metadata {
        name      = var.name
        namespace = var.env.namespace
        labels = {
          app     = var.name
          deploy  = "http"
          version = var.tag == "latest" ? "latest-${local.timestamptag}" : var.tag
        }
      }

      spec {
        container {
          name = var.name

          image = "${var.image}:${var.tag}"

          image_pull_policy = var.env.type == "dev" ? "Never" : "IfNotPresent"

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

          liveness_probe {
            http_get {
              path = var.setup.health
              port = var.setup.port
            }

            initial_delay_seconds = 5
            period_seconds        = 5
          }
        }
      }
    }
  }
}
