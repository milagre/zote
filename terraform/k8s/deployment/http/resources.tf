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

          liveness_probe {
            http_get {
              path = var.setup.health
              port = var.setup.port
            }

            initial_delay_seconds = 5
            period_seconds        = 15
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
      }
    }
  }
}

resource "kubernetes_service" "service" {
  metadata {
    name      = var.name
    namespace = var.namespace
    annotations = merge(
    )
  }

  spec {
    type = "ClusterIP"

    port {
      port        = 80
      target_port = var.setup.port
      protocol    = "TCP"
      name        = "http"
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
      host = var.internal.private_hostname

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

locals {
  ingress_hostnames = concat(
    [
      var.internal.public_hostname,
    ],
    var.internal.veneer_hostnames,
  )
}

resource "kubernetes_ingress_v1" "public_nginx" {
  count = 1 # TODO: conditional

  metadata {
    name      = "${var.name}-nginx-public"
    namespace = var.namespace
    annotations = {
      "kubernetes.io/ingress.class"    = "nginx"
      "cert-manager.io/cluster-issuer" = "letsencrypt"
    }
  }

  wait_for_load_balancer = false

  spec {
    ingress_class_name = "nginx"

    dynamic "rule" {
      for_each = local.ingress_hostnames

      content {
        host = rule.value

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

    dynamic "tls" {
      for_each = var.env.is_local ? [] : [1]

      content {
        hosts       = local.ingress_hostnames
        secret_name = "${var.name}-tls"
      }
    }
  }
}


resource "kubernetes_ingress_v1" "ngrok" {
  count = var.env.is_local ? 1 : 0

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
      host = var.internal.public_hostname

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
