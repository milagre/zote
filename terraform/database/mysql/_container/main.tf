variable "env" {}
variable "name" {}
variable "namespace" {}
variable "primary_profile" {}
variable "replica_profile" {}
variable "database" {}
variable "username" {}

locals {
  images = {
    xtrabackup = "bitnami/percona-xtrabackup:latest"
    mysql      = "mysql:8"
    init       = "bash:5"
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

resource "kubernetes_config_map" "config" {
  metadata {
    name      = "cfg-${var.name}"
    namespace = var.namespace

    labels = {
      app = var.name
    }
  }

  data = {
    "primary.cnf" = <<-EOF
        [mysqld]
        log-bin
    EOF

    "replica.cnf" = <<-EOF
        [mysqld]
        super-read-only
    EOF
  }
}

resource "kubernetes_secret" "password" {
  metadata {
    name      = "cfg-${var.name}"
    namespace = var.namespace

    labels = {
      app = var.name
    }
  }

  data = {
    MYSQL_PASSWORD = local.password
  }
}

resource "kubernetes_service" "svc" {
  metadata {
    name      = var.name
    namespace = var.namespace

    labels = {
      app = var.name
    }
  }

  spec {
    port {
      name = "mysql"
      port = 3306
    }

    selector = {
      app = var.name
    }
  }
}

resource "kubernetes_stateful_set" "sts" {
  metadata {
    name      = var.name
    namespace = var.namespace
  }

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
            name = kubernetes_config_map.config.metadata[0].name
          }
        }

        init_container {
          name  = "init-mysql"
          image = local.images.init
          command = [
            "bash",
            "-c",
            <<-EOF
                set -ex

                # Generate mysql server-id from pod ordinal index.
                [[ `hostname` =~ -([0-9]+)$ ]] || exit 1
                ordinal=$${BASH_REMATCH[1]}
                echo [mysqld] > /mnt/conf.d/server-id.cnf

                # Add an offset to avoid reserved server-id=0 value.
                echo server-id=$((100 + $ordinal)) >> /mnt/conf.d/server-id.cnf

                # Copy appropriate files from config-map to conf.d.
                if [[ $ordinal -eq 0 ]]; then
                    cp /mnt/config-map/primary.cnf /mnt/conf.d/
                else
                    cp /mnt/config-map/replica.cnf /mnt/conf.d/
                fi
            EOF
          ]

          volume_mount {
            name       = "conf"
            mount_path = "/mnt/conf.d"
          }

          volume_mount {
            name       = "config-map"
            mount_path = "/mnt/config-map"
          }
        }

        container {
          name  = "mysql"
          image = local.images.mysql

          port {
            name           = "mysql"
            container_port = 3306
          }

          dynamic "env" {
            for_each = {
              MYSQL_ALLOW_EMPTY_PASSWORD = "yes"
              MYSQL_ROOT_HOST            = "127.0.0.1"
              MYSQL_DATABASE             = var.database
              MYSQL_USER                 = var.username
            }
            content {
              name  = env.key
              value = env.value
            }
          }

          env_from {
            secret_ref {
              name = kubernetes_secret.password.metadata[0].name
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
            mount_path = "/var/lib/mysql"
            sub_path   = "mysql"
          }

          volume_mount {
            name       = "conf"
            mount_path = "/etc/mysql/conf.d"
          }

          liveness_probe {
            exec {
              command = ["mysqladmin", "ping"]
            }

            initial_delay_seconds = 30
            timeout_seconds       = 5
            period_seconds        = 10
          }

          readiness_probe {
            exec {
              command = ["mysqladmin", "ping"]
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
  value     = 3306
  sensitive = false
}

output "password" {
  value     = local.password
  sensitive = true
}
