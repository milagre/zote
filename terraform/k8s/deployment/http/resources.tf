module "deployment" {
  source = "../_deployment"
  env    = var.env

  namespace = var.namespace
  name      = var.name
  type      = "http"

  image   = var.image
  tag     = var.tag
  profile = var.profile

  cmd  = var.cmd
  args = var.args

  conf  = var.conf
  files = var.files

  http_liveness_probe = {
    path = var.setup.health
    port = var.setup.port
    freq = var.setup.freq
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
  count = var.internal.public_hostname == null ? 0 : 1

  metadata {
    name      = "${var.name}-nginx-public"
    namespace = var.namespace
    annotations = {
      "kubernetes.io/ingress.class"    = "nginx"
      "cert-manager.io/cluster-issuer" = "letsencrypt-http01"
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
