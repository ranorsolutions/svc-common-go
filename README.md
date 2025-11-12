# svc-common-go

A shared service library for Go microservices at **Ranor Solutions**, providing consistent service setup, gRPC/HTTP integration, configuration management, and shared tooling.
It is designed to work seamlessly alongside [`http-common-go`](https://github.com/ranorsolutions/http-common-go) to create a unified service architecture that scales cleanly across domains.

---

## 📦 Overview

The `svc-common-go` package provides reusable components to standardize the lifecycle of Go-based services.
It abstracts away repetitive boilerplate so each service can focus purely on business logic, not infrastructure.

---

## 🧱 Architecture

```
svc-common-go/
├── pkg/
│   ├── firebase_service/   # Optional Firebase client abstraction
│   ├── grpc/               # gRPC server setup utilities
│   ├── http/               # HTTP server setup via Gin
│   ├── server/             # Unified multiplexer (gRPC + HTTP) via cmux
│   ├── service/            # Core service struct (DB, connections, logger)
│   └── types/              # Common type definitions for handlers and routes
├── Makefile                # Development and testing utilities
├── go.mod                  # Module definition
└── README.md               # This file
```

---

## ⚙️ Features

- **Unified Service Bootstrapping**
  Simplifies consistent initialization across microservices.

- **gRPC + HTTP Multiplexing**
  Uses [`cmux`](https://github.com/soheilhy/cmux) to serve both protocols on the same port.

- **Service Dependency Injection**
  Loads downstream service gRPC connections automatically from environment variables.

- **Integrated Logging**
  Uses the structured logging from [`http-common-go/pkg/log/logger`](https://github.com/ranorsolutions/http-common-go).

- **Optional Firebase Integration**
  Provides a wrapper for authentication and messaging without forcing Firebase as a dependency.

- **Simple Error Handling**
  Standardized `HandleErr` helper for consistent JSON error responses in Gin handlers.

---

## 🧰 Installation

```bash
go get github.com/ranorsolutions/svc-common-go
```

If you’re developing locally with both libraries:
```bash
make use_dev
```
This replaces the `http-common-go` dependency with a local path reference.

---

## 🚀 Usage

### Basic Example
```go
package main

import (
    "log"

    "github.com/ranorsolutions/svc-common-go/pkg/service"
    "github.com/ranorsolutions/svc-common-go/pkg/server"
)

func main() {
    svc, err := service.New()
    if err != nil {
        log.Fatalf("failed to initialize service: %v", err)
    }

    srv := server.New(svc, "v1")
    srv.Run()
}
```

### HTTP Routing
To attach HTTP handlers:
```go
import (
    "github.com/gin-gonic/gin"
    "github.com/ranorsolutions/svc-common-go/pkg/service"
    "github.com/ranorsolutions/svc-common-go/pkg/types"
)

func main() {
    svc, _ := service.New()
    svc.HTTPHandlers = []*types.HTTPHandler{
        {
            Type: "GET",
            Path: "/health",
            Handler: []gin.HandlerFunc{
                func(c *gin.Context) {
                    c.JSON(200, gin.H{"status": "ok"})
                },
            },
        },
    }
}
```

## 🧪 Testing

Run all tests:
```bash
make test
```

Generate coverage report:
```bash
make coverage
```

Run race detector:
```bash
make race
```

---

## 🧹 Development Utilities

| Command         | Description                                  |
| --------------- | -------------------------------------------- |
| `make fmt`      | Format and tidy Go code                      |
| `make test`     | Run all unit tests                           |
| `make race`     | Run tests with race detector                 |
| `make coverage` | Generate HTML coverage report                |
| `make lint`     | Run static analysis (requires golangci-lint) |
| `make clean`    | Clean test cache and artifacts               |
| `make use_dev`  | Use local `http-common-go` dependency        |
| `make use_prod` | Restore remote `http-common-go` dependency   |

---

## 🧠 Extending the Service

The `svc-common-go` package is designed to grow with your system architecture — it’s not just a template, it’s a foundation.  
You can extend it by defining new service modules, background workers, and cross-cutting functionality without rewriting boilerplate.

### 🔌 Adding a Custom Module

A new domain or subsystem (for example, “Billing”) can be added under `pkg/billing`:

```bash
svc-common-go/
└── pkg/
    ├── billing/
    │   ├── billing.go
    │   └── billing_test.go
```

**Example implementation:**
```go
package billing

import (
    "context"
    "fmt"
    "github.com/ranorsolutions/svc-common-go/pkg/service"
)

type BillingService struct {
    svc *service.Service
}

func New(svc *service.Service) *BillingService {
    return &BillingService{svc: svc}
}

func (b *BillingService) ProcessInvoice(ctx context.Context, invoiceID string) error {
    b.svc.Logger.Info("Processing invoice: %s", invoiceID)
    // your business logic here
    return nil
}
```

You can then initialize and attach it from your main service:
```go
svc, _ := service.New()
billing := billing.New(svc)
svc.Services["billing"] = billing
```

---

### 🧵 Running Background Workers

Long-running or scheduled tasks can live alongside HTTP and gRPC handlers without blocking the main server.

Example pattern:
```go
func startBackgroundTasks(svc *service.Service) {
    go func() {
        ticker := time.NewTicker(1 * time.Hour)
        defer ticker.Stop()

        for range ticker.C {
            svc.Logger.Info("Running scheduled cleanup...")
            // perform cleanup or sync
        }
    }()
}
```

You can call this from your entrypoint right before starting the server:
```go
func main() {
    svc, _ := service.New()
    startBackgroundTasks(svc)
    server.New(svc, "v1").Run()
}
```

---

### 🧠 Best Practices

| Category              | Recommendation                                                                                                           |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| **Configuration**     | Keep all configuration in environment variables and access them via `os.Getenv` or a config loader.                      |
| **Logging**           | Always use `svc.Logger` for structured, consistent logging.                                                              |
| **gRPC Clients**      | Use `SERVICE_DEPS` to inject downstream connections and re-use `svc.ServiceConnections`.                                 |
| **Database Access**   | Prefer context-aware operations (`db.QueryContext`, `db.ExecContext`, etc.).                                             |
| **Caching**           | For shared cache logic, integrate `pkg/cache` from [`http-common-go`](https://github.com/ranorsolutions/http-common-go). |
| **Tracing & Metrics** | Extend via OpenTelemetry or Prometheus middleware — both integrate cleanly with the service struct.                      |

---

### ⚡ Example Project Layout

A production service using both `svc-common-go` and `http-common-go` might look like:

```
myservice/
├── cmd/
│   └── main.go
├── pkg/
│   ├── domain/
│   │   ├── user/
│   │   │   ├── user.go
│   │   │   ├── user_handler.go
│   │   │   └── user_repo.go
│   └── ...
├── internal/
│   ├── config/
│   ├── migration/
│   └── worker/
├── go.mod
├── Makefile
└── README.md
```

The `cmd/main.go` bootstraps `svc-common-go`, wires dependencies, and starts the unified HTTP/gRPC server.

---

### 🧩 Integration with Other Common Packages

| Package                                                              | Purpose                                                         |
| -------------------------------------------------------------------- | --------------------------------------------------------------- |
| [`http-common-go`](https://github.com/ranorsolutions/http-common-go) | Middleware, logging, caching, and common utilities              |
| `svc-common-go`                                                      | Service initialization, routing, and unified protocol serving   |
| Future: `msg-common-go`                                              | Messaging (Kafka, SNS, SQS, etc.) integration layer             |
| Future: `obs-common-go`                                              | Observability (tracing, metrics, logging aggregation) utilities |

---

## 🤝 Contributing

Contributions are welcome!  
Please follow standard Go formatting (`go fmt ./...`) and ensure all tests pass before submitting a pull request:

```bash
make fmt
make test
```

---

## 📄 License

This project is licensed under the **MIT License**.
See the full text in [LICENSE](./LICENSE).

---

## 👩‍💻 Maintainer

**Abigail Ranson**
Maintainer — [Ranor Solutions](https://ranorsolutions.com)
© 2025 Abigail Ranson. All rights reserved.

