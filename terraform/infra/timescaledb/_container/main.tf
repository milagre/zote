variable "env" {} # tflint-ignore: terraform_unused_declarations
variable "namespace" {}
variable "name" {}
variable "profile" {}

variable "user" {
  type = string
}

variable "database" {
  type = string
}

locals {
  name = "timescaledb-${var.name}"
}

module "profile" {
  source = "./../../../structs/profile"

  profile = var.profile
}

resource "random_password" "password" {
  length  = 32
  special = false

  min_numeric = 8
  min_lower   = 8
  min_upper   = 8
}

resource "kubernetes_persistent_volume_claim_v1" "timescale_pvc" {
  metadata {
    name      = "${local.name}-pvc"
    namespace = var.namespace
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

resource "kubernetes_stateful_set_v1" "timescaledb" {
  metadata {
    name      = local.name
    namespace = var.namespace
    labels = {
      app = local.name
    }
  }

  spec {
    service_name = local.name
    replicas     = 1

    selector {
      match_labels = {
        app = local.name
      }
    }

    template {
      metadata {
        labels = {
          app = local.name
        }
      }

      spec {
        container {
          name  = "timescaledb"
          image = "timescale/timescaledb:latest-pg17"

          env {
            name  = "POSTGRES_USER"
            value = var.user
          }

          env {
            name  = "POSTGRES_PASSWORD"
            value = random_password.password.result
          }

          env {
            name  = "POSTGRES_DB"
            value = var.database
          }

          env {
            name  = "PGDATA"
            value = "/var/lib/postgresql/data/pgdata"
          }

          port {
            container_port = 5432
          }

          volume_mount {
            name       = "timescale-storage"
            mount_path = "/var/lib/postgresql/data"
          }

          resources {
            limits = {
              cpu    = module.profile.cpu_cores.max
              memory = "${module.profile.mem_mb.max}Mi"
            }
            requests = {
              cpu    = module.profile.cpu_cores.min
              memory = "${module.profile.mem_mb.min}Mi"
            }
          }
        }

        volume {
          name = "timescale-storage"
          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim.timescale_pvc.metadata[0].name
          }
        }
      }
    }
  }
}

resource "kubernetes_service_v1" "timescaledb" {
  metadata {
    name      = local.name
    namespace = var.namespace
    labels = {
      app = local.name
    }
  }

  spec {
    selector = {
      app = local.name
    }

    port {
      protocol    = "TCP"
      port        = 5432
      target_port = 5432
    }

    type = "ClusterIP"
  }
}

resource "kubernetes_secret_v1" "timescale_secret" {
  metadata {
    name      = "${local.name}-secret"
    namespace = var.namespace
  }

  data = {
    PGHOST     = local.name
    PGPORT     = "5432"
    PGDATABASE = var.database
    PGUSER     = var.user
    PGPASSWORD = random_password.password.result
  }
}

output "scheme" {
  value = "postgresql"
}

output "host" {
  value = local.name
}

output "port" {
  value = 5432
}

output "user" {
  value = var.user
}

output "pass" {
  value     = random_password.password.result
  sensitive = true
}

output "database" {
  value = var.database
}
