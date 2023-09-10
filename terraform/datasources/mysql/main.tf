variable "env" {}
variable "name" {}
variable "namespace" {}

locals {
  name = "mysql-${var.name}"
  cfg  = "MYSQL_${upper(var.name)}"
  images = {
    xtrabackup = "bitnami/percona-xtrabackup:latest"
    mysql      = "mysql:8"
  }
}

resource "kubernetes_config_map" "config" {
  metadata {
    name      = "cfg-${local.name}"
    namespace = var.namespace

    labels = {
      app = local.name
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

resource "kubernetes_service" "svc" {
  metadata {
    name      = local.name
    namespace = var.namespace

    labels = {
      app = local.name
    }
  }

  spec {
    port {
      name = "mysql"
      port = 3306
    }

    selector = {
      app = local.name
    }

    cluster_ip = "None"
  }
}

resource "kubernetes_stateful_set" "sts" {
  metadata {
    name      = local.name
    namespace = var.namespace
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
          image = local.images.mysql
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

                # Copy appropriate conf.d files from config-map to emptyDir.
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

          env {
            name  = "MYSQL_ALLOW_EMPTY_PASSWORD"
            value = "1"
          }

          resources {
            requests = {
              cpu = "500m"

              memory = "1Gi"
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
              command = ["mysql", "-h", "127.0.0.1", "-e", "SELECT 1"]
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

resource "kubernetes_config_map" "client" {
  metadata {
    name      = local.name
    namespace = var.namespace

    labels = {
      app = local.name
    }
  }

  data = {
    "${local.cfg}_HOST" = "${local.name}.svc.cluster.local"
    "${local.cfg}_PORT" = "3306"
    "${local.cfg}_USER" = "root"
    "${local.cfg}_PASS" = ""
  }
}
