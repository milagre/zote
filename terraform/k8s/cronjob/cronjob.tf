
locals {
  timestamptag = replace(timestamp(), "/[-:TZ]/", "")
  tag          = var.tag == "latest" ? "latest-${local.timestamptag}" : var.tag
}

resource "kubernetes_cron_job_v1" "job" {
  metadata {
    name      = var.name
    namespace = var.namespace
    labels = {
      // app     = var.name
      // version = local.tag
    }
  }

  spec {

    concurrency_policy            = "Replace"
    failed_jobs_history_limit     = 5
    schedule                      = var.schedule
    starting_deadline_seconds     = 10
    successful_jobs_history_limit = 10

    job_template {
      metadata {
        name = var.name
        labels = {
          // app     = var.name
          // version = local.tag
        }
      }

      spec {
        backoff_limit              = 2
        ttl_seconds_after_finished = 3600

        template {
          metadata {
            name = var.name
            labels = {
              // app     = var.name
              // version = local.tag
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
                  cpu    = module.profile.cpu_cores.max
                  memory = "${module.profile.mem_mb.max}M"
                }
                requests = {
                  cpu    = module.profile.cpu_cores.min
                  memory = "${module.profile.mem_mb.min}M"
                }
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
  }
}
