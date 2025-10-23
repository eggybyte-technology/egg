# Egg Framework Port Allocation

This document describes the port allocation strategy for all services in the Egg Framework testing environment.

## Port Allocation Strategy

To avoid port conflicts when running multiple services, we use the following strategy:

- **Infrastructure Services**: Use standard ports (MySQL, Jaeger, OTLP, etc.)
- **Application Services**: Use standard ports internally (8080, 8081, 9091) but map to different host ports

## Port Mapping Table

### Infrastructure Services

| Service | Port | Description |
|---------|------|-------------|
| MySQL | 3306 | Database server |
| Jaeger UI | 16686 | Tracing UI |
| Jaeger Collector | 14268 | Trace ingestion |
| OTLP gRPC | 4317 | OpenTelemetry gRPC endpoint |
| OTLP HTTP | 4318 | OpenTelemetry HTTP endpoint |
| Prometheus Exporter | 8889 | Metrics export from OTEL collector |

### Minimal Connect Service

| Container Port | Host Port | Purpose |
|----------------|-----------|---------|
| 8080 | 8080 | HTTP/Connect API |
| 8081 | 8081 | Health check endpoint |
| 9091 | 9091 | Prometheus metrics |

**Access URLs:**
- API: http://localhost:8080
- Health: http://localhost:8081/health
- Metrics: http://localhost:9091/metrics

### User Service

| Container Port | Host Port | Purpose |
|----------------|-----------|---------|
| 8080 | 8082 | HTTP/Connect API |
| 8081 | 8083 | Health check endpoint |
| 9091 | 9092 | Prometheus metrics |

**Access URLs:**
- API: http://localhost:8082
- Health: http://localhost:8083/health
- Metrics: http://localhost:9092/metrics

## Port Conflict Resolution

If you encounter port conflicts when starting services, use the provided cleanup script:

```bash
./scripts/cleanup-ports.sh
```

This script will:
1. Identify processes using required ports
2. Attempt to terminate conflicting processes
3. Verify all ports are free

## Adding New Services

When adding a new service to the deployment:

1. **Choose Non-Conflicting Host Ports**: Use ports that don't conflict with existing services
2. **Use Standard Internal Ports**: Keep container internal ports as 8080, 8081, 9091 for consistency
3. **Update Documentation**: Add your service to this file
4. **Update Cleanup Script**: Add your ports to `scripts/cleanup-ports.sh`
5. **Update Tests**: Add health checks to `scripts/test.sh`

### Example Port Allocation for a New Service

```yaml
new-service:
  container_name: egg-new-service
  ports:
    - "8084:8080"   # HTTP API
    - "8085:8081"   # Health check
    - "9093:9091"   # Metrics
```

## Troubleshooting

### Port Already in Use

If you see errors like:
```
Error: bind: address already in use
```

**Solution:**
1. Run the cleanup script: `./scripts/cleanup-ports.sh`
2. Or manually find and kill the process:
   ```bash
   lsof -ti:8080 | xargs kill -9
   ```
3. Or stop all containers:
   ```bash
   cd deploy && docker-compose down --remove-orphans
   ```

### Containers Not Stopping

If containers don't stop cleanly:

```bash
docker ps -a | grep egg- | awk '{print $1}' | xargs docker rm -f
```

### Port Check

To check if a port is available:

```bash
lsof -ti:8080
# No output = port is free
# PID number = port is in use
```

## Best Practices

1. **Always Clean Before Start**: Run cleanup before starting services in development
2. **Use Make Targets**: Prefer `make test-examples` which handles cleanup automatically
3. **Check Logs**: If services fail to start, check `docker-compose logs` for details
4. **Sequential Testing**: Run one set of tests at a time to avoid resource conflicts

## Docker Compose Commands

Common commands for managing services:

```bash
# Start all services
cd deploy && docker-compose up -d

# Stop all services
cd deploy && docker-compose down

# View logs
cd deploy && docker-compose logs -f [service-name]

# Restart a service
cd deploy && docker-compose restart [service-name]

# Check service status
cd deploy && docker-compose ps
```

## References

- [docker-compose.yaml](./docker-compose.yaml) - Service definitions
- [cleanup-ports.sh](../scripts/cleanup-ports.sh) - Port cleanup script
- [test.sh](../scripts/test.sh) - Integration testing script

