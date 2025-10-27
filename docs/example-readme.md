# egg/connectx

## Overview

`connectx` is the unified RPC layer for egg microservices.
It provides a production-ready stack of Connect interceptors
that ensure consistent timeout control, observability, and
structured error mapping across all services.

## Key Features

- Per-RPC and global timeout enforcement
- Structured logging with request and trace correlation
- Prometheus and OpenTelemetry metrics integration
- Error mapping from core/errors to Connect and HTTP
- Flexible interceptor chaining for platform and business logic
- Configurable slow-request logging and payload metrics

## Dependencies

Layer: **L3 (Runtime Communication Layer)**  
Depends on: `core`, `logx`, `configx`, `obsx`

## Installation

```bash
go get go.eggybyte.com/egg/connectx@latest
````

## Basic Usage

```go
import (
    "github.com/bufbuild/connect-go"
    "go.eggybyte.com/egg/connectx"
    "net/http"
)

mux := http.NewServeMux()
opts := connectx.Options{DefaultTimeoutMs: 1000}
path, handler := demoV1connect.NewGreeterHandler(
    NewGreeterService(),
    connect.WithInterceptors(connectx.DefaultInterceptors(opts)...),
)
mux.Handle(path, handler)
```

## Configuration Options

| Option                 | Type    | Description                                |
| ---------------------- | ------- | ------------------------------------------ |
| `DefaultTimeoutMs`     | `int64` | Default per-RPC timeout (ms)               |
| `AllowClientDownscale` | `bool`  | Allow client to shorten timeout via header |
| `SlowRequestMs`        | `int64` | Threshold for slow-request warning logs    |
| `EmitPayloadMetrics`   | `bool`  | Collect payload size metrics               |
| `EnableRetry`          | `bool`  | Enable client-side retries (clientx only)  |

## Observability

Each RPC emits standard metrics and tracing spans:

* `rpc.server.requests_total{service,method,code}`
* `rpc.server.duration_ms_bucket{le}`
* `rpc.client.retries_total`

All logs follow the unified single-line `logfmt` standard.

## Stability

Stable since **v0.1.0**
Backward-compatible API changes are guaranteed for minor version updates.

## License

Licensed under the Apache License 2.0.