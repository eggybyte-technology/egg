<!-- ff5aa76b-da6f-4930-940c-3c264d417d82 836fbf77-a4fa-400b-9d72-c717d1eb2925 -->
# Egg monorepo implementation plan (client-go + otelconnect)

## Decisions

- Repo/module root: `github.com/eggybyte-technology/egg`
- Go version: use local toolchain; generate `go` directive via go CLI
- Create `go.work` and all `go.mod` via go CLI only
- k8sx: integrate `k8s.io/client-go` now (ConfigMap watch + discovery)
- connectx: use `connectrpc.com/otelconnect` for trace/metrics; add our recovery/logging/identity/error-mapping interceptors
- No `examples/`

## Directory layout

- `core/` (stable, zero deps): `log/`, `errors/`, `identity/`, `utils/`
- `runtimex/`: runtime lifecycle, ports, graceful shutdown (no transport)
- `connectx/`: Connect interceptors + header-based identity, bind helper
- `obsx/`: OpenTelemetry/Prometheus provider bootstrap
- `k8sx/`: ConfigMap watch (informers), service discovery (client-go, EndpointSlice)
- `storex/` (optional, thin for now): interfaces + health checks; adapters later

## High-level steps

1) Initialize workspace modules

- From repo root: create `go.work` with `core/`, `runtimex/`, `connectx/`, `obsx/`, `k8sx/`, `storex/`
- For each module: `go mod init github.com/eggybyte-technology/egg/<module>`

2) `core` public API (stable)

- `core/log`: define `Logger` interface and lightweight kv helpers (`Str/Int/Dur`)
- `core/errors`: `Code` (string), `E` struct, `New`, `Wrap`, `CodeOf`
- `core/identity`: `UserInfo`, `RequestMeta`, `WithUser/UserFrom`, `WithMeta/MetaFrom`
- `core/utils`: minimal helpers (e.g., retry/backoff stub, time helpers)

3) `runtimex` runtime

- Types: `Service`, `Endpoint`, `HTTPOptions{Addr,H2C,Mux}`, `RPCOptions{Addr}`, `Options{Logger,HTTP,RPC,Health,Metrics,ShutdownTimeout}`
- `Run(ctx, svcs, opts)`: start HTTP (h2c optional), health, metrics servers; manage lifecycle/shutdown; structured logging using `core/log`

4) `obsx` provider

- `Options{ServiceName,ServiceVersion,OTLPEndpoint,EnableRuntimeMetrics,ResourceAttrs,TraceSamplerRatio}`
- `NewProvider(ctx, Options) (*Provider, error)` sets up `TracerProvider` + `MeterProvider` (OTLP/gRPC); `Shutdown(ctx)`

5) `connectx`

- `HeaderMapping` with default Higress headers; `Options{Logger,Otel,*Headers,WithRequestBody,WithResponseBody,SlowRequestMillis,PayloadAccounting}`
- `DefaultInterceptors(o Options) []connect.Interceptor` composing:
  - `otelconnect` (trace/metrics)
  - recovery (panic to internal error)
  - logging (structured, fields per guide; optional payload sizes)
  - identity injector (extract from headers → `core/identity`)
  - error mapper (`core/errors.Code` → `connect.Code` → HTTP)
- `Bind(mux *http.ServeMux, path string, h http.Handler)` utility

6) `k8sx` (client-go based)

- `WatchConfigMap(ctx, name string, o WatchOptions, onUpdate func(map[string]string))` using shared informer; supports in-cluster and kubeconfig
- `Resolve(ctx, service string, kind ServiceKind) ([]string /* host:port */, error)`
  - For `headless`: list `discovery.k8s.io/v1` `EndpointSlice` or fallback to `Endpoints`; collect `ip:port`
  - For `clusterip`: fetch `Service`, pick port by name `http` or first TCP port; return `service.namespace.svc:port` (or ClusterIP:port)

7) `storex` (thin)

- Minimal registry: register connections by name; `Ping(ctx)` aggregator for health; no GORM adapters yet

8) Wire dependencies via go CLI

- `connectx`: `connectrpc.com/connect`, `connectrpc.com/otelconnect`
- `obsx`: `go.opentelemetry.io/otel/...` (sdk, exporters/otlp/otlptracegrpc, resource), `google.golang.org/grpc`
- `runtimex`: `golang.org/x/net/http2/h2c`, `prometheus` client for metrics export
- `k8sx`: `k8s.io/client-go`, `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/kube-openapi` (transitive)
- `storex`: none or `database/sql` only for now

9) Build validation

- `go build ./...` at repo root (with `go work`)

## Key API snippets (representative)

```go
// core/log
package log

type Logger interface {
  With(kv ...any) Logger
  Debug(msg string, kv ...any)
  Info(msg string, kv ...any)
  Warn(msg string, kv ...any)
  Error(err error, msg string, kv ...any)
}
```



```go
// runtimex
package runtimex

type Service interface{ Start(ctx context.Context) error; Stop(ctx context.Context) error }

type HTTPOptions struct{ Addr string; H2C bool; Mux *http.ServeMux }

type Options struct{
  Logger log.Logger
  HTTP *HTTPOptions
  RPC  *RPCOptions
  Health, Metrics *Endpoint
  ShutdownTimeout time.Duration
}
```



```go
// connectx (interceptor composition sketch)
func DefaultInterceptors(o Options) []connect.Interceptor {
  // order: recovery → otelconnect → identity → error map → logging
  // return []connect.Interceptor{ ... }
}
```

## Commands to be executed (later, by go CLI only)

- From repo root `/Users/fengguangyao/eggybyte/projects/go/egg`:
  - `go work init ./core ./runtimex ./connectx ./obsx ./k8sx ./storex`
- For each module:
  - `cd <module> && go mod init github.com/eggybyte-technology/egg/<module>`
- Add deps (examples):
  - `cd connectx && go get connectrpc.com/connect connectrpc.com/otelconnect`
  - `cd obsx && go get go.opentelemetry.io/otel/... google.golang.org/grpc`
  - `cd runtimex && go get golang.org/x/net/http2/h2c github.com/prometheus/client_golang/prometheus`
  - `cd k8sx && go get k8s.io/client-go k8s.io/api k8s.io/apimachinery`
- Build:
  - `cd .. && go build ./...`

## Notes

- RBAC: `k8sx` will require list/watch on ConfigMaps, Services, EndpointSlices
- OTel: ensure `OTEL_EXPORTER_OTLP_ENDPOINT` reachable; default sampler ratio configurable
- No concrete logger shipped; apps provide an implementation of `core/log.Logger`

### To-dos

- [ ] Init go.work and per-module go.mod via go CLI
- [ ] Implement core: log, errors, identity, utils
- [ ] Implement runtimex: types and Run with HTTP/health/metrics
- [ ] Implement obsx provider (OTLP gRPC) and Shutdown
- [ ] Implement connectx interceptors (otelconnect, recovery, logging, identity, errors)
- [ ] Implement k8sx ConfigMap watch and Resolve via client-go
- [ ] Add storex interfaces and health registry (no DB yet)
- [ ] Add module dependencies with go get by module
- [ ] Build all modules with go work: go build ./...