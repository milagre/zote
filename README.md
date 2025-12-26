# Zote

![image](https://github.com/milagre/zote/assets/1005028/6bfb6b50-e181-4d60-840d-95bc242c97e4)

## Project Overview

**Zote** is a comprehensive Go framework providing a set of independant but interconnected utility libraries for common web application needs. Pick and choose what you like, ignore the rest. But if you decide to use more, know that every module is designed to play well with the others (and may use the others under the hood).

## Go Library

### Go Module Structure

The framework consists of ~20 independent modules, each providing specific functionality:

**CLI & Configuration:**

- **[zcmd](go/zcmd)** - CLI application framework with environment-based configuration

**Utilities:**

- **[zhttpclient](go/zhttpclient)** - HTTP client helpers with more intelligent defaults
- **[zlog](go/zlog)** - Interface-based logging adaptable to other log providers.
- **[zsig](go/zsig)** - Unix signal helpers
- **[zencodeio](go/zencodeio)** - Reader and Writer adapter for Marshallers (e.g. json)
- **[ztime](go/ztime)** - Enhanced date and time types with JSON/SQL support
- **[zwarn](go/zwarn)** - Warning system parallel to errors (non-fatal errors)

**Database & ORM:**

- **[zsql](go/zsql)** - The missing low-level interfaces from database/sql.
- **[zorm](go/zorm)** - High-level ORM with generic CRUD operations

**API Framework:**

- **[zapi](go/zapi)** - HTTP API server framework

**Messaging:**

- **[zamqp](go/zamqp)** - RabbitMQ/AMQP integration and process framework

**Caching:**

- **[zcache](go/zcache)** - Cache abstraction with read-through pattern

**Time-Series Data:**

- **[zts](go/zts)** - Time-series database abstractions
  - **ztimescaledb** - TimescaleDB integration for PostgreSQL
  - **zinfluxdb** - InfluxDB client wrapper

**Helpers:**

- **[zfunc](go/zfunc)** - Functional programming constructs missing from the std library (maps, slices)
- **[zreflect](go/zreflect)** - Reflection helpers

## Terraform Library

The `terraform/` directory provides reusable infrastructure modules for kubernetes-based deployments designed to support development environments very closely mirroring production ones, and a standardized deployment style designed to play well with zcmd. A pluggable cloud provider setup decouples provider-specific configuration from core system design.

**Cloud Providers:**

- digialocean
- aws (coming soon)

**Kubernetes Modules:**

- `k8s/deployment` - A kubernetes Deployment with a few standardized setup style for different types of processes.
- `k8s/job` - A kubernetes Job with the same standardized setup style
- `k8s/cronjob` - A kubernetes CronJob with the same standardized setup style

**Database Modules:**

- `database/mysql` - MySQL deployment supporting both containerized (Kubernetes) and cloud (DigitalOcean) modes
  - Outputs ConfigMap and Secret for client configuration
