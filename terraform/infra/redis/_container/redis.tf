locals {
  shard_ids = {
    for idx in range(var.shards) :
    idx => substr("abcdefghijklmnopqrstuvwxyz", idx, 1)
  }
}

resource "kubernetes_service" "redis" {
  metadata {
    name      = "redis-${var.name}"
    namespace = var.namespace
    labels = {
      app = "redis-${var.name}"
    }
  }

  spec {
    port {
      port = 6379
    }

    selector = {
      app = "redis-${var.name}"
    }
  }
}

resource "kubernetes_config_map" "redis_conf" {
  count = var.shards

  metadata {
    name      = "cfg-redis-${var.name}-${local.shard_ids[count.index]}"
    namespace = var.namespace
    labels = {
      app = "redis-${var.name}"
    }
  }

  data = {
    "redis.conf" = file("${path.module}/redis.conf")
  }
}

resource "kubernetes_config_map" "redis_scripts" {
  metadata {
    name      = "cfg-redis-${var.name}-scripts"
    namespace = var.namespace
    labels = {
      app = "redis-${var.name}"
    }
  }

  data = {
    "update-nodes.sh" = "${file("${path.module}/update-nodes.sh")}"
  }
}

resource "kubernetes_stateful_set" "redis" {
  count = var.shards

  metadata {
    name      = "redis-${var.name}-${local.shard_ids[count.index]}"
    namespace = var.namespace
  }

  spec {
    replicas = var.replicas + 1

    selector {
      match_labels = {
        app = "redis-${var.name}"
      }
    }

    service_name = kubernetes_service.redis.metadata[0].name

    template {
      metadata {
        labels = {
          app = "redis-${var.name}"
        }
      }

      spec {
        container {
          name              = "redis"
          image             = "redis:${var.ver}"
          image_pull_policy = "IfNotPresent"

          port {
            container_port = 6379
            name           = "client"
          }

          port {
            container_port = 16379
            name           = "cluster"
          }

          command = ["/etc/scripts/update-nodes.sh", "redis-server", "/etc/redis/redis.conf"]

          env {
            name = "POD_IP"
            value_from {
              field_ref {
                field_path = "status.podIP"
              }
            }
          }

          volume_mount {
            name       = "data"
            mount_path = "/data"
          }

          volume_mount {
            name       = "config"
            mount_path = "/etc/redis"
          }

          volume_mount {
            name       = "scripts"
            mount_path = "/etc/scripts"
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
            name = kubernetes_config_map.redis_conf[count.index].metadata[0].name
          }
        }

        volume {
          name = "scripts"
          config_map {
            name         = kubernetes_config_map.redis_scripts.metadata[0].name
            default_mode = "0755"
          }
        }

        volume {
          name = "shared"
          empty_dir {

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
  }
}

resource "kubernetes_job" "cluster" {
  metadata {
    name      = "redis-${var.name}-cluster"
    namespace = var.namespace
  }

  spec {
    template {
      metadata {
        name = "redis-${var.name}-cluster"
      }

      spec {
        container {
          name              = "redis"
          image             = "redis:${var.ver}"
          image_pull_policy = "IfNotPresent"

          command = flatten([
            "redis-cli",
            "-h", "redis-${var.name}-a-0.redis-${var.name}",
            "-p", "6379",
            "--cluster", "create",
            flatten([
              for node in range(var.replicas + 1) :
              [
                for id in values(local.shard_ids) :
                "redis-${var.name}-${id}-${node}.redis-${var.name}:6379"
              ]
            ]),
            "--cluster-replicas", "${var.replicas}",
            "--cluster-yes",
          ])
        }
      }
    }
  }
}
