module "deployment" {
  source = "../_deployment"
  env    = var.env

  namespace = var.namespace
  name      = var.name
  type      = "proc"

  image   = var.image
  tag     = var.tag
  profile = var.profile

  cmd  = var.cmd
  args = var.args

  conf  = var.conf
  files = var.files
}

resource "kubernetes_service" "service" {
  metadata {
    name      = var.name
    namespace = var.namespace
    annotations = merge(
    )
  }

  spec {
    cluster_ip = "None"

    selector = {
      app    = var.name
      deploy = "proc"
    }

    port {
      port        = 80
      target_port = 80
      protocol    = "TCP"
    }
  }
}
