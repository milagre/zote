
resource "kubernetes_service_account_v1" "rabbitmq" {
  metadata {
    name      = local.name
    namespace = var.namespace
  }
}

resource "kubernetes_role_v1" "rabbitmq" {
  metadata {
    name      = local.name
    namespace = var.namespace
  }

  rule {
    verbs      = ["get"]
    api_groups = [""]
    resources  = ["endpoints"]
  }

  rule {
    verbs      = ["create"]
    api_groups = [""]
    resources  = ["events"]
  }
}

resource "kubernetes_role_binding_v1" "rabbitmq" {
  metadata {
    name      = local.name
    namespace = var.namespace
  }

  subject {
    kind      = "ServiceAccount"
    name      = kubernetes_service_account_v1.rabbitmq.metadata[0].name
    namespace = var.namespace
  }

  role_ref {
    api_group = "rbac.authorization.k8s.io"
    kind      = "Role"
    name      = kubernetes_role_v1.rabbitmq.metadata[0].name
  }
}
