variable "env" {} # tflint-ignore: terraform_unused_declarations
variable "name" {}
variable "namespace" {}
variable "ver" {}
variable "primary_profile" {}
variable "replica_profile" {}
variable "database" {}
variable "username" {}

locals {
  port = 5432
  images = {
    postgresql = "postgres:${var.ver}-alpine"
  }
}

module "primary_profile" {
  source = "./../../../structs/profile"

  profile = var.primary_profile
}

module "replica_profile" {
  source = "./../../../structs/profile"

  profile = var.replica_profile
}

resource "random_password" "password" {
  length  = 64
  special = false

  min_numeric = 8
  min_lower   = 8
  min_upper   = 8

  override_special = "$%&*()-_=+[]{}<>:?"
}

locals {
  password = random_password.password.result
}

resource "kubernetes_config_map_v1" "config" {
  metadata {
    name      = "cfg-${var.name}"
    namespace = var.namespace

    labels = {
      app = var.name
    }
  }

  data = {
    "postgresql.conf" = <<-EOF
    EOF
  }
}

resource "kubernetes_secret_v1" "password" {
  metadata {
    name      = "cfg-${var.name}"
    namespace = var.namespace

    labels = {
      app = var.name
    }
  }

  data = {
    POSTGRES_PASSWORD = local.password
  }
}

resource "kubernetes_service_v1" "svc" {
  metadata {
    name      = var.name
    namespace = var.namespace

    labels = {
      app = var.name
    }
  }

  spec {
    port {
      name = "postgresql"
      port = local.port
    }

    selector = {
      app = var.name
    }
  }
}

resource "kubernetes_stateful_set_v1" "sts" {
  metadata {
    name      = var.name
    namespace = var.namespace
  }

  wait_for_rollout = false

  spec {
    service_name = var.name
    replicas     = 1

    selector {
      match_labels = {
        app = var.name
      }
    }

    template {
      metadata {
        labels = {
          app = var.name
        }
      }

      spec {
        volume {
          name = "conf"
          empty_dir {}
        }

        volume {
          name = "config-map"

          config_map {
            name = kubernetes_config_map_v1.config.metadata[0].name
          }
        }

        container {
          name  = "postgresql"
          image = local.images.postgresql

          port {
            name           = "postgresql"
            container_port = local.port
          }

          dynamic "env" {
            for_each = {
              POSTGRES_DATABASE = var.database
              POSTGRES_USER     = var.username
            }
            content {
              name  = env.key
              value = env.value
            }
          }

          env_from {
            secret_ref {
              name = kubernetes_secret_v1.password.metadata[0].name
            }
          }

          resources {
            requests = {
              cpu    = module.primary_profile.cpu_cores.min
              memory = "${module.primary_profile.mem_mb.min}M"
            }
            limits = {
              cpu    = module.primary_profile.cpu_cores.max
              memory = "${module.primary_profile.mem_mb.max}M"
            }
          }

          volume_mount {
            name       = "data"
            mount_path = "/var/lib/postgresql/data/pgdata"
          }

          volume_mount {
            name       = "conf"
            mount_path = "/etc/postgresql/postgresql.conf"
            sub_path   = "postgresql.conf"
          }

          liveness_probe {
            exec {
              command = ["pg_isready", "-U", var.username]
            }

            initial_delay_seconds = 30
            timeout_seconds       = 5
            period_seconds        = 10
          }

          readiness_probe {
            exec {
              command = ["pg_isready", "-U", var.username]
            }

            initial_delay_seconds = 5
            timeout_seconds       = 1
            period_seconds        = 2
          }
        }
      }
    }

    volume_claim_template {
      metadata {
        name = "data"
      }

      spec {
        access_modes = ["ReadWriteOnce"]

        resources {
          requests = {
            storage = "10Gi"
          }
        }
      }
    }
  }
}

output "username" {
  value     = var.username
  sensitive = false
}

output "hostname" {
  value     = "${var.name}.${var.namespace}.svc.cluster.local"
  sensitive = false
}

output "port" {
  value     = local.port
  sensitive = false
}

output "password" {
  value     = local.password
  sensitive = true
}
