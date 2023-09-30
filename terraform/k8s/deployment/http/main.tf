variable "env" {}
variable "name" {}
variable "namespace" {}
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
variable "files" {
  type = object({
    configmaps = map(string),
  })
  default = {
    configmaps = {},
  }
}
variable "ngrok" {
  type = object({
    enabled = bool
    domain  = string
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
      deploy  = "http"
      version = local.tag
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
        namespace = var.namespace
        labels = {
          app     = var.name
          deploy  = "http"
          version = local.tag
        }
      }

      spec {
        container {
          name = var.name

          image = "${var.image}:${var.tag}"

          image_pull_policy = var.env.is_dev ? "Never" : "IfNotPresent"

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

          dynamic "env" {
            for_each = var.conf.values
            content {
              name  = env.key
              value = env.value
            }
          }
        }

        dynamic "volume" {
          for_each = var.files.configmaps

          content {
            name = volume.key
            config_map {
              name = volume.value
            }
          }
        }
      }
    }
  }
}


resource "kubernetes_service" "service" {
  metadata {
    name      = var.name
    namespace = var.namespace
  }

  spec {

    type = var.env.lb_type
    port {
      port        = 80
      target_port = var.setup.port
      protocol    = "TCP"
    }

    selector = {
      app    = var.name
      deploy = "http"
    }
  }

}

resource "kubernetes_ingress_v1" "private_nginx" {
  metadata {
    name      = "${var.name}-nginx-private"
    namespace = var.namespace
    annotations = {
      "kubernetes.io/ingress.class" = "nginx"
    }
  }

  wait_for_load_balancer = false

  spec {
    ingress_class_name = "nginx"

    rule {
      host = "${var.name}.${var.namespace}.localhost.localdomain"

      http {
        path {
          path      = "/"
          path_type = "Prefix"

          backend {
            service {
              name = kubernetes_service.service.metadata[0].name
              port {
                number = 80
              }
            }
          }
        }
      }
    }
  }
}

resource "kubernetes_ingress_v1" "public_nginx" {
  count = 1 # TODO: conditional

  metadata {
    name      = "${var.name}-nginx-public"
    namespace = var.namespace
    annotations = {
      "kubernetes.io/ingress.class" = "nginx"
    }
  }

  wait_for_load_balancer = false

  spec {
    ingress_class_name = "nginx"

    rule {
      host = "${var.name}.${var.namespace}.${var.ngrok.domain}"

      http {
        path {
          path      = "/"
          path_type = "Prefix"

          backend {
            service {
              name = kubernetes_service.service.metadata[0].name
              port {
                number = 80
              }
            }
          }
        }
      }
    }
  }
}


resource "kubernetes_ingress_v1" "ngrok" {
  count = var.ngrok == null ? 0 : 1

  metadata {
    name      = "${var.name}-ngrok"
    namespace = var.namespace
    annotations = {
      "kubernetes.io/ingress.class" = "ngrok"
    }
  }

  spec {
    ingress_class_name = "ngrok"

    rule {
      host = "${var.name}.${var.namespace}.${var.ngrok.domain}"

      http {
        path {
          path      = "/"
          path_type = "Prefix"

          backend {
            service {
              name = kubernetes_service.service.metadata[0].name
              port {
                number = 80
              }
            }
          }
        }
      }
    }
  }
}
