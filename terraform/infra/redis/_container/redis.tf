resource "kubernetes_service" "redis" {
  metadata {
    name      = "redis-${var.name}"
    namespace = var.namespace
    labels = {
      app = var.name
    }
  }

  spec {
    port {
      port = 6379
    }

    selector = {
      app = var.name
    }

    cluster_ip = "None"
  }
}

resource "kubernetes_config_map" "redis" {
  metadata {
    name      = "cfg-redis-${var.name}"
    namespace = var.namespace
    labels = {
      app = var.name
    }
  }

  data = {
    "primary.conf" = "${file("${path.module}/primary.conf")}"
    "replica.conf" = <<EOF
        slaveof redis-${var.name}-0.redis-${var.name}.redis 6379
        maxmemory ${floor(module.profile.mem_mb.max * 0.9)}mb
        maxmemory-policy allkeys-lru
        timeout 0
        dir /data
    EOF
  }
}

resource "kubernetes_stateful_set" "redis" {
  metadata {
    name      = "redis-${var.name}"
    namespace = var.namespace
    annotations = {
    }
  }

  spec {
    replicas = module.profile.num.max

    selector {
      match_labels = {
        app = var.name
      }
    }

    service_name = kubernetes_service.redis.metadata[0].name

    template {
      metadata {
        labels = {
          app = var.name
        }

        annotations = {
        }
      }

      spec {
        init_container {
          name              = "init"
          image             = "redis:${var.ver}"
          image_pull_policy = "IfNotPresent"
          command           = ["/bin/bash", "-c", ]
          args = [<<-EOF
            set -ex
            # Generate redis server-id from pod ordinal index.
            [[ `hostname` =~ -([0-9]+)$ ]] || exit 1
            ordinal=$${BASH_REMATCH[1]}
            # Copy appropriate redis config files from config-map to respective directories.
            if [[ $ordinal -eq 0 ]]; then
                cp /mnt/primary.conf /etc/redis.conf
            else
                cp /mnt/replica.conf /etc/redis.conf
            fi
            EOF
          ]

          volume_mount {
            name       = "claim"
            mount_path = "/etc"
          }
          volume_mount {
            name       = "config"
            mount_path = "/mnt/"
          }
        }

        container {
          name              = "redis"
          image             = "redis:${var.ver}"
          image_pull_policy = "IfNotPresent"

          port {
            container_port = 6379
            name           = "redis"
          }
          command = ["redis-server", "/etc/redis.conf"]

          volume_mount {
            name       = "data"
            mount_path = "/data"
          }

          volume_mount {
            name       = "claim"
            mount_path = "/etc"
          }
          resources {
            limits = {
              cpu    = module.profile.cpu_cores.max
              memory = "${module.profile.mem_mb.max}M"
            }

            requests = {
              cpu    = module.profile.cpu_cores.min
              memory = "${module.profile.mem_mb.min}M"
            }
          }
        }
        volume {
          name = "config"
          config_map {
            name = kubernetes_config_map.redis.metadata[0].name
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
            storage = "${floor(module.profile.mem_mb.max * 1.1)}M"
          }
        }
      }
    }
    volume_claim_template {
      metadata {
        name = "claim"
      }
      spec {
        access_modes = ["ReadWriteOnce"]
        resources {
          requests = {
            storage = "${floor(module.profile.mem_mb.max * 1.1)}M"
          }
        }
      }
    }
  }
}
