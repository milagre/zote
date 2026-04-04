module "mysql" {
  source    = "../../../../../terraform/database/mysql"
  env       = var.env
  namespace = var.namespace
  name      = "core"
  ver       = var.config.data.service.db.version
  database  = "example"
  username  = "example"
  container = var.config.data.service.db.container
  cloud     = null
}

module "rabbitmq" {
  source    = "../../../../../terraform/infra/rabbitmq"
  env       = var.env
  namespace = var.namespace
  name      = "core"
  ver       = var.config.data.service.rabbitmq.version
  container = var.config.data.service.rabbitmq.container
  setup = {
    users = [
      { name = "api", tags = [] },
      { name = "workers", tags = [] },
    ]
    vhosts = [
      { name = "/", users = ["api", "workers"] },
    ]
  }
}

module "api" {
  source  = "../../../../../terraform/k8s/deployment"
  env     = var.env
  profile = var.config.data.service.api.profile

  name      = "api"
  namespace = var.namespace

  image = "${try(var.config.data.service.registry, "")}example/service"
  tag   = var.config.data.service.version
  args  = ["api"]

  http = {
    port   = 80
    health = "/_health"
    freq   = var.env.is_dev ? 300 : null
  }

  public_domains = try(var.config.data.service.public_domains, [])

  prometheus_monitored = try(var.config.data.service.prometheus.podmonitors.enabled, false)

  conf = {
    values = merge(var.config.env_vars, {
      "DEMO_LISTEN_PORT" = "80"
    })
    configmaps = [
      module.mysql.k8s.configmap,
      module.rabbitmq.k8s.amqp.configmap,
      module.rabbitmq.k8s.users["api"].configmap,
    ]
    secrets = [
      module.mysql.k8s.secret,
      module.rabbitmq.k8s.users["api"].secret,
    ]
  }

  depends_on = [
    module.mysql,
    module.rabbitmq,
  ]
}

module "deleter" {
  source  = "../../../../../terraform/k8s/deployment"
  env     = var.env
  profile = var.config.data.service.deleter.profile

  name      = "deleter"
  namespace = var.namespace

  image = "${try(var.config.data.service.registry, "")}example/service"
  tag   = var.config.data.service.version
  args  = ["deleter"]

  prometheus_monitored = try(var.config.data.service.prometheus.podmonitors.enabled, false)

  conf = {
    values = merge(var.config.env_vars, {
      "DEMO_CONCURRENCY" = tostring(var.config.data.service.deleter.concurrency)
    })
    configmaps = [
      module.mysql.k8s.configmap,
      module.rabbitmq.k8s.amqp.configmap,
      module.rabbitmq.k8s.users["workers"].configmap,
    ]
    secrets = [
      module.mysql.k8s.secret,
      module.rabbitmq.k8s.users["workers"].secret,
    ]
  }

  depends_on = [
    module.mysql,
    module.rabbitmq,
  ]
}
