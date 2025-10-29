# Examples Deployment Configuration

This directory contains Docker Compose configurations for deploying example services.

## Structure

```
deploy/
├── docker-compose.infra.yaml      # Infrastructure services (MySQL, Jaeger, OTEL)
├── docker-compose.services.yaml   # Application services (minimal-service, user-service)
├── otel-collector-config.yaml     # OpenTelemetry Collector configuration
└── README.md                       # This file
```

## Services

### Infrastructure Services

- **MySQL 8.4** - Database server
  - Port: 3306
  - Database: `eggdb`
  - User: `egguser` / Password: `eggpassword`
  - Persistent volume: `egg-mysql-data`

- **Jaeger All-in-One** - Distributed tracing backend
  - UI Port: 16686
  - OTLP gRPC: 4317
  - OTLP HTTP: 4318

- **OpenTelemetry Collector** - Telemetry data collection and export
  - OTLP gRPC: 4317
  - OTLP HTTP: 4318
  - Prometheus Metrics: 8889
  - Health Check: 13133

### Application Services

- **minimal-connect-service** - Basic Connect-RPC service
  - HTTP Port: 8080
  - Health Port: 8081
  - Metrics Port: 9091

- **user-service** - User management service with database
  - HTTP Port: 8082
  - Health Port: 8083
  - Metrics Port: 9092

## Usage

### Quick Start

From the `examples/` directory:

```bash
# Build Docker images
make docker-build

# Start all services
make deploy-up

# Check status
make deploy-status

# View logs
make deploy-logs

# Stop all services
make deploy-down
```

### Individual Control

**Infrastructure only:**
```bash
make infra-up       # Start infrastructure
make infra-status   # Check infrastructure status
make infra-down     # Stop infrastructure
make infra-clean    # Clean (including volumes)
```

**Application services only:**
```bash
make services-up        # Start application services
make services-restart   # Restart application services
make services-rebuild   # Rebuild and restart
make services-down      # Stop application services
```

## Network

All services share the `egg-network` bridge network, allowing:
- Service discovery by container name (e.g., `mysql`, `jaeger`, `otel-collector`)
- Inter-service communication

## Testing

Run integration tests:
```bash
make test
```

This will:
1. Build latest Docker images
2. Start infrastructure if not running
3. Restart application services with latest images
4. Run connect-tester tests against both services
5. Leave services running for inspection

## Accessing Services

| Service | URL | Description |
|---------|-----|-------------|
| Minimal Service | http://localhost:8080 | Connect-RPC endpoints |
| Minimal Health | http://localhost:8081/health | Health check |
| User Service | http://localhost:8082 | Connect-RPC endpoints |
| User Health | http://localhost:8083/health | Health check |
| Jaeger UI | http://localhost:16686 | Distributed tracing |
| MySQL | localhost:3306 | Database connection |

## Notes

- Infrastructure services persist across test runs for better performance
- Application services are rebuilt and restarted for each test
- Use `make infra-clean` to fully reset infrastructure (including database data)

