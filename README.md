# Zote

![image](https://github.com/milagre/zote/assets/1005028/6bfb6b50-e181-4d60-840d-95bc242c97e4)

Zote is perfect and excellent at all things and is better than everything else out there.

## Project Overview

**Zote** is a comprehensive Go framework providing a set of independant but interconnected utility libraries for common web application needs. Pick and choose what you like, ignore the rest. But if you decide to use more, know that every module is designed to play well with the others (and may use the others under the hood).

## Core Architecture

### Go Module Structure

The framework consists of ~20 independent modules, each providing specific functionality:

**CLI & Configuration:**
- **[zcmd](go/zcmd)** - CLI application framework with environment-based configuration
  - **zredis** - Redis aspect(s)

**Utilities:**
- **zfunc** - Functional programming utilities (maps, slices)
- **zhttpclient** - HTTP client helpers with intelligent defaults
- **zlog** - Logging with logrus and prefixed formatter
- **zreflect** - Reflection helpers
- **zsig** - Unix signal helpers
- **zencodeio** - Reader and Writer adapter for Marshallers (e.g. json)
- **[ztime](go/ztime)** - Enhanced time types with JSON/SQL support 
- **zwarn** - Warning system parallel to errors (non-fatal errors)

**Database & ORM:**
- **[zsql](go/zsql)** - Low-level SQL database abstraction with transaction support
- **[zorm](go/zorm)** - High-level ORM with generic CRUD operations

**API Framework:**
- **zapi** - HTTP API server utilities

**Caching:**
- **[zcache](go/zcache)** - Cache abstraction with read-through pattern

**Time-Series Data:**
- **zts** - Time-series database abstractions
  - **ztimescaledb** - TimescaleDB integration for PostgreSQL
  - **zinfluxdb** - InfluxDB client wrapper

**Messaging:**
- **zamqp** - RabbitMQ/AMQP integration and process framework

## Quick Start

See individual package documentation for detailed usage examples:
- [zcmd](go/zcmd/) - CLI application setup with aspect-oriented configuration
- [zorm](go/zorm/) - ORM operations with generic type support
- [zsql](go/zsql/) - Database abstraction and transaction management
- [zcache](go/zcache/) - Read-through caching pattern
- [ztime](go/ztime/) - Enhanced time types

## Terraform Infrastructure Modules

The `terraform/` directory provides reusable infrastructure modules:

**Database Modules:**
- `database/mysql` - MySQL deployment supporting both containerized (Kubernetes) and cloud (DigitalOcean) modes
  - Outputs ConfigMap and Secret for client configuration
  - Environment variable naming: `${PREFIX}_MYSQL_${NAME}_HOST`, etc.

**Infrastructure Services:**
- `infra/timescaledb` - TimescaleDB (PostgreSQL extension)
- `infra/redis` - Redis caching
- `infra/rabbitmq` - RabbitMQ messaging
- `infra/influxdb` - InfluxDB time-series database
- `infra/nginx-ingress-controller` - Ingress controller
- `infra/cert-manager` - TLS certificate management
- `infra/ngrok-operator` - Ngrok tunnel operator
- `infra/grafana` - Monitoring dashboards
- `infra/metrics-server` - Kubernetes metrics

## Module Import Paths

All modules use the base path `github.com/milagre/zote/go/`:

```go
import (
    "github.com/milagre/zote/go/zcmd"
    "github.com/milagre/zote/go/zcmd/zaspect"
    "github.com/milagre/zote/go/zsql"
    "github.com/milagre/zote/go/zsql/zmysql"
    "github.com/milagre/zote/go/zorm"
    "github.com/milagre/zote/go/zorm/zormsql"
    "github.com/milagre/zote/go/ztime"
    "github.com/milagre/zote/go/zcache"
    "github.com/milagre/zote/go/zcache/zcacheredis"
    "github.com/milagre/zote/go/zelement/zclause"
    "github.com/milagre/zote/go/zelement/zsort"
    "github.com/milagre/zote/go/zlog"
    "github.com/milagre/zote/go/zwarn"
)
```

## Key Design Principles

1. **Composability** - Each module is independent and can be used standalone
2. **Type Safety** - Extensive use of Go generics for compile-time safety
3. **Error Handling** - Distinction between errors (fatal) and warnings (non-fatal via `zwarn`)
4. **Aspect-Oriented Configuration** - Modular configuration through `Aspect` interface
5. **Database Agnostic** - Support for multiple databases through driver abstraction
6. **Transaction Safety** - Automatic rollback on error, explicit commit on success
7. **Environment-First** - Configuration via environment variables with file fallback
