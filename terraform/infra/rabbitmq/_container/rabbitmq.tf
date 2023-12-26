locals {
  users = concat(var.setup.users, [{
    "name" : "admin",
    "tags" : ["administrator", "management"]
  }])
}

resource "random_bytes" "salts" {
  for_each = toset([for user in local.users : user.name])

  length = 4
}

resource "random_password" "passwords" {
  for_each = toset([for user in local.users : user.name])

  length  = 32
  special = false
}

data "external" "password_hashes" {
  for_each = toset([for user in local.users : user.name])

  program = [
    "bash",
    "${path.module}/pwhash.sh",
    random_bytes.salts[each.key].base64,
    random_password.passwords[each.key].result,
  ]
}

locals {
  definitions = {
    /*
    bindings          = [],
    exchanges         = [],
    global_parameters = [],
    parameters        = [],
    policies          = [],
    queues            = [],
    topic_permissions = [],
    */
    permissions = flatten([
      for vhost in var.setup.vhosts : concat([
        for user in vhost.users :
        {
          user      = user,
          vhost     = "${trimprefix(vhost.name, "/")}",
          configure = ".*",
          read      = ".*",
          write     = ".*",
        }
        ], [
        {
          user      = "admin",
          vhost     = "${trimprefix(vhost.name, "/")}",
          configure = ".*",
          read      = ".*",
          write     = ".*",
        }
    ])]),
    users = [
      for user in local.users :
      {
        hashing_algorithm = "rabbit_password_hashing_sha512",
        name              = user.name,
        password_hash     = data.external.password_hashes[user.name].result.hash,
        tags              = user.tags,
      }
    ],
    vhosts = concat([
      for vhost in var.setup.vhosts :
      {
        name = "${trimprefix(vhost.name, "/")}",
      }
      ], [
      {
        name = "/"
      }
    ])
  }

}

resource "random_password" "password" {
  length  = 32
  special = false
}

resource "random_password" "erlang_cookie" {
  length  = 32
  special = false
}

resource "kubernetes_config_map" "config" {
  metadata {
    name      = "cfg-${local.name}"
    namespace = var.namespace
  }

  data = {
    "enabled_plugins" = "[rabbitmq_peer_discovery_k8s, rabbitmq_management, rabbitmq_prometheus].\n"

    "definitions.json" : jsonencode(local.definitions)

    "rabbitmq.conf" = <<-END
      cluster_formation.peer_discovery_backend = k8s
      cluster_formation.k8s.host = kubernetes.default.svc.cluster.local
      cluster_formation.k8s.address_type = hostname
      cluster_formation.k8s.service_name = ${local.name}-headless

      definitions.import_backend = local_filesystem
      definitions.local.path = /etc/rabbitmq/definitions.json

      queue_master_locator=min-masters

      loopback_users.admin = true
      default_vhost = /
      default_user = admin
      default_pass = ${random_password.passwords["admin"].result}
      default_permissions.configure = .*
      default_permissions.read = .*
      default_permissions.write = .*
      default_user_tags.administrator = true
      default_user_tags.management = true
    END

    "username" = "rabbitmq"
  }
}

resource "kubernetes_secret" "config" {
  metadata {
    name      = "cfg-${local.name}"
    namespace = var.namespace
  }

  data = {
    "password"      = random_password.password.result
    "erlang_cookie" = random_password.erlang_cookie.result
  }
}

resource "kubernetes_stateful_set" "rabbitmq" {
  metadata {
    name      = local.name
    namespace = var.namespace
  }

  spec {
    replicas = module.profile.num.min

    selector {
      match_labels = {
        app = local.name
      }
    }

    template {
      metadata {
        name = local.name

        labels = {
          app = local.name
        }
      }

      spec {
        init_container {
          name  = "rabbitmq-config"
          image = "busybox:1.32.0"
          command = [
            "sh",
            "-c",
            <<-END
              cp /tmp/rabbitmq/rabbitmq.conf /etc/rabbitmq/rabbitmq.conf && echo '' >> /etc/rabbitmq/rabbitmq.conf; \
              cp /tmp/rabbitmq/enabled_plugins /etc/rabbitmq/enabled_plugins; \
              cp /tmp/rabbitmq/definitions.json /etc/rabbitmq/definitions.json
            END
          ]

          volume_mount {
            name       = "config"
            mount_path = "/tmp/rabbitmq"
          }

          volume_mount {
            name       = "config-rw"
            mount_path = "/etc/rabbitmq"
          }
        }

        container {
          name  = "rabbitmq"
          image = "rabbitmq:${var.ver}"

          port {
            name           = "amqp"
            container_port = 5672
            protocol       = "TCP"
          }

          port {
            name           = "management"
            container_port = 15672
            protocol       = "TCP"
          }

          port {
            name           = "prometheus"
            container_port = 15692
            protocol       = "TCP"
          }

          port {
            name           = "epmd"
            container_port = 4369
            protocol       = "TCP"
          }

          env {
            name = "RABBITMQ_DEFAULT_PASS"

            value_from {
              secret_key_ref {
                name = kubernetes_secret.config.metadata[0].name
                key  = "password"
              }
            }
          }

          env {
            name = "RABBITMQ_DEFAULT_USER"

            value_from {
              config_map_key_ref {
                name = kubernetes_config_map.config.metadata[0].name
                key  = "username"
              }
            }
          }

          env {
            name = "RABBITMQ_ERLANG_COOKIE"

            value_from {
              secret_key_ref {
                name = kubernetes_secret.config.metadata[0].name
                key  = "erlang_cookie"
              }
            }
          }

          volume_mount {
            name       = "config-rw"
            mount_path = "/etc/rabbitmq"
          }

          volume_mount {
            name       = "data"
            mount_path = "/var/lib/rabbitmq/mnesia"
          }

          liveness_probe {
            exec {
              command = ["rabbitmq-diagnostics", "status"]
            }

            initial_delay_seconds = 60
            timeout_seconds       = 15
            period_seconds        = 60
          }

          readiness_probe {
            exec {
              command = ["rabbitmq-diagnostics", "ping"]
            }

            initial_delay_seconds = 10
            timeout_seconds       = 10
            period_seconds        = 10
          }

          resources {
            requests = {
              cpu    = module.profile.cpu_cores.min
              memory = "${module.profile.mem_mb.min}M"
            }
            limits = {
              cpu    = module.profile.cpu_cores.max
              memory = "${module.profile.mem_mb.max}M"
            }
          }
        }

        volume {
          name = "config"

          config_map {
            name = kubernetes_config_map.config.metadata[0].name

            items {
              key  = "enabled_plugins"
              path = "enabled_plugins"
            }

            items {
              key  = "rabbitmq.conf"
              path = "rabbitmq.conf"
            }

            items {
              key  = "definitions.json"
              path = "definitions.json"
            }
          }
        }

        volume {
          name = "config-rw"
          empty_dir {
          }
        }

        volume {
          name = "data"

          persistent_volume_claim {
            claim_name = "data"
          }
        }

        security_context {
          run_as_user  = 999
          run_as_group = 999
          fs_group     = 999
        }

        service_account_name = kubernetes_service_account.rabbitmq.metadata[0].name
      }
    }

    volume_claim_template {
      metadata {
        name      = "data"
        namespace = var.namespace
      }

      spec {
        access_modes = ["ReadWriteOnce"]

        resources {
          requests = {
            storage = "2Gi"
          }
        }

        storage_class_name = "standard"
      }
    }

    service_name = "${local.name}-headless"
  }
}

resource "kubernetes_service" "client" {
  metadata {
    name      = local.name
    namespace = var.namespace

    labels = {
      app = local.name
    }
  }

  spec {
    port {
      name     = "http"
      protocol = "TCP"
      port     = 15672
    }

    port {
      name     = "prometheus"
      protocol = "TCP"
      port     = 15692
    }

    port {
      name     = "amqp"
      protocol = "TCP"
      port     = 5672
    }

    selector = {
      app = local.name
    }

    type = var.env.lb_type
  }

  lifecycle {
    ignore_changes = [
      metadata[0].annotations["kubernetes.digitalocean.com/load-balancer-id"]
    ]
  }
}

resource "kubernetes_service" "headless" {
  metadata {
    name      = "${local.name}-headless"
    namespace = var.namespace
  }

  spec {
    port {
      name        = "epmd"
      protocol    = "TCP"
      port        = 4369
      target_port = "4369"
    }

    port {
      name        = "cluster-rpc"
      protocol    = "TCP"
      port        = 25672
      target_port = "25672"
    }

    selector = {
      app = local.name
    }

    cluster_ip       = "None"
    type             = "ClusterIP"
    session_affinity = "None"
  }
}
