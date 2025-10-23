# Egg Framework Testing Guide

This guide describes the testing infrastructure for the Egg Framework and how to run various test scenarios.

## Overview

The Egg Framework includes comprehensive testing capabilities:

- **Unit Tests**: Test individual modules independently
- **Integration Tests**: Test example services with full infrastructure
- **CLI Tests**: Validate the `egg` CLI tool functionality
- **Production Tests**: Test with remote module dependencies

## Test Infrastructure

### Services

The test environment includes the following services:

| Service | Purpose | Ports |
|---------|---------|-------|
| MySQL | Database for user-service | 3306 |
| Jaeger | Distributed tracing UI | 16686, 14268 |
| OTLP Collector | Telemetry aggregation | 4317 (gRPC), 4318 (HTTP), 8889 (metrics) |
| minimal-connect-service | Basic Connect service example | 8080 (HTTP), 8081 (health), 9091 (metrics) |
| user-service | Full-featured user management service | 8082 (HTTP), 8083 (health), 9092 (metrics) |

### Port Allocation

All services use non-conflicting ports. See [deploy/PORTS.md](../deploy/PORTS.md) for detailed port allocation.

## Running Tests

### Quick Start

```bash
# Run all tests
make test-all

# Run only example service tests
make test-examples

# Run CLI tests (local modules)
make test-cli

# Run CLI tests (remote modules)
make test-cli-production

# Run unit tests for all modules
make test
```

### Manual Port Cleanup

If you encounter port conflicts:

```bash
# Check and free required ports
make deploy-ports

# Or run the cleanup script directly
./scripts/cleanup-ports.sh
```

### Manual Service Management

```bash
# Start all services
make deploy-up

# Stop all services
make deploy-down

# Restart services
make deploy-restart

# View logs
make deploy-logs

# Check service health
make deploy-health

# Clean everything
make deploy-clean
```

## Test Workflow

### Example Services Test (test-examples)

The `make test-examples` command performs the following steps:

1. **Cleanup Phase**
   - Stop all existing docker-compose services
   - Remove all egg-related containers
   - Wait for containers to fully stop
   - Run port cleanup script to free required ports

2. **Build Phase**
   - Pull eggybyte-go-alpine base image
   - Build minimal-connect-service binary and image
   - Build user-service binary and image

3. **Deploy Phase**
   - Start infrastructure services (MySQL, Jaeger, OTLP Collector)
   - Wait for MySQL to be healthy
   - Start application services (minimal-service, user-service)
   - Wait for services to be ready (15 seconds)

4. **Test Phase**
   - Check minimal-service health endpoint (localhost:8081/health)
   - Check user-service health endpoint (localhost:8083/health)
   - Test Connect endpoints using connect-tester
   - Verify service functionality

5. **Cleanup Phase**
   - Stop all services
   - Display logs if any tests failed

### Expected Test Duration

- **Cleanup**: ~5 seconds
- **Build**: ~10-30 seconds (depending on cache)
- **Deploy**: ~30-40 seconds (waiting for MySQL and services)
- **Test**: ~10 seconds
- **Total**: ~1-2 minutes

## Common Issues and Solutions

### Issue 1: Port Already in Use

**Symptoms:**
```
Error: bind: address already in use
```

**Solution:**
```bash
# Option 1: Run automated cleanup
make deploy-ports

# Option 2: Manual cleanup
lsof -ti:8080 | xargs kill -9
lsof -ti:8082 | xargs kill -9

# Option 3: Stop all containers
cd deploy && docker-compose down --remove-orphans
```

### Issue 2: Container Won't Stop

**Symptoms:**
```
Error: container is still running
```

**Solution:**
```bash
# Force remove all egg containers
docker ps -a | grep egg- | awk '{print $1}' | xargs docker rm -f

# Or use the clean target
make deploy-clean
```

### Issue 3: OTEL Collector Configuration Error

**Symptoms:**
```
'exporters' unknown type: "jaeger"
```

**Solution:**
This has been fixed in the latest configuration. The OTEL Collector now uses the `otlp/jaeger` exporter instead of the deprecated `jaeger` exporter. If you see this error:

1. Pull the latest changes
2. Verify `deploy/otel-collector-config.yaml` uses `otlp/jaeger` exporter
3. Restart the services

### Issue 4: macOS bash Compatibility

**Symptoms:**
```
declare: -A: invalid option
```

**Solution:**
This has been fixed in the latest `scripts/cleanup-ports.sh`. The script now uses bash 3.x compatible array syntax instead of associative arrays.

### Issue 5: MySQL Connection Timeout

**Symptoms:**
```
dial tcp: connect: connection refused
```

**Solution:**
```bash
# Wait longer for MySQL to be ready
# The test script waits 15 seconds, but you can increase this

# Or manually verify MySQL is ready
docker exec egg-mysql mysqladmin ping -h localhost -uroot -prootpassword
```

## Test Debugging

### View Service Logs

```bash
# All services
cd deploy && docker-compose logs

# Specific service
cd deploy && docker-compose logs user-service

# Follow logs in real-time
cd deploy && docker-compose logs -f minimal-service
```

### Check Service Health

```bash
# Minimal service
curl http://localhost:8081/health

# User service
curl http://localhost:8083/health

# Jaeger UI
curl http://localhost:16686

# Prometheus metrics
curl http://localhost:8889/metrics
```

### Test Connect Endpoints

```bash
# Using the connect-tester tool
cd scripts/connect-tester
go run main.go http://localhost:8080 minimal-service
go run main.go http://localhost:8082 user-service

# Using curl (requires proper Connect headers)
curl -X POST http://localhost:8082/user.v1.UserService/CreateUser \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","name":"Test User"}'
```

### Inspect Containers

```bash
# List all egg containers
docker ps -a | grep egg-

# Inspect a container
docker inspect egg-user-service

# Execute commands in a container
docker exec -it egg-user-service sh

# Check container logs
docker logs egg-minimal-service
```

## Continuous Integration

### GitHub Actions

The project includes GitHub Actions workflows for:

- **CI**: Run unit tests, lint, and build on every push
- **Release**: Publish binaries and Docker images on tag

### Local Pre-commit Checks

Run these before committing:

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests
make test

# Run all quality checks
make quality
```

## Best Practices

### 1. Clean Environment

Always start with a clean environment:

```bash
make deploy-clean
make deploy-ports
```

### 2. Sequential Testing

Run one test suite at a time to avoid resource conflicts:

```bash
# Good
make test
make test-cli
make test-examples

# Avoid running multiple test commands simultaneously
```

### 3. Check Logs on Failure

When tests fail, always check the logs:

```bash
# Quick log check
make deploy-logs

# Or detailed analysis
cd deploy && docker-compose logs > test-failure.log
```

### 4. Verify Port Availability

Before starting services, verify ports are free:

```bash
# Check specific port
lsof -ti:8080

# Check all required ports
make deploy-ports
```

### 5. Use Make Targets

Prefer using Make targets over direct script calls:

```bash
# Good
make test-examples

# Avoid
./scripts/test.sh examples
```

## Performance Optimization

### Speed Up Builds

1. **Use Docker layer caching**:
   - The base image is pulled once and cached
   - Go module downloads are cached

2. **Parallel builds**:
   - Build services in parallel when possible
   - Current implementation builds sequentially

3. **Skip tests during development**:
   ```bash
   # Just build without testing
   make docker-all
   
   # Start services without rebuilding
   cd deploy && docker-compose up -d
   ```

## Writing New Tests

### Adding a New Service

1. Update `deploy/docker-compose.yaml`:
   - Add service definition
   - Assign non-conflicting ports
   - Add health check

2. Update `scripts/cleanup-ports.sh`:
   - Add your ports to the PORTS array

3. Update `scripts/test.sh`:
   - Add health check for your service
   - Add functional tests if needed

4. Update documentation:
   - Add service to `deploy/PORTS.md`
   - Update this testing guide

### Example: Adding New Service

```yaml
# In deploy/docker-compose.yaml
new-service:
  image: new-service:latest
  container_name: egg-new-service
  environment:
    HTTP_PORT: :8080
    HEALTH_PORT: :8081
  ports:
    - "8084:8080"  # Non-conflicting host port
    - "8085:8081"
  networks:
    - egg-network
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
    interval: 30s
    timeout: 10s
    retries: 3
```

```bash
# In scripts/cleanup-ports.sh (add to PORTS array)
"8084:New Service HTTP"
"8085:New Service Health"
```

```bash
# In scripts/test.sh (add health check)
print_info "Testing new service health..."
if curl -f http://localhost:8085/health > /dev/null 2>&1; then
    print_success "New service health check passed"
else
    print_warning "New service health check failed"
    test_failed=1
fi
```

## References

- [Architecture Guide](./guide.md)
- [Port Allocation](../deploy/PORTS.md)
- [Docker Compose Configuration](../deploy/docker-compose.yaml)
- [Build Scripts](../scripts/build.sh)
- [Test Scripts](../scripts/test.sh)
- [Deployment Scripts](../scripts/deploy.sh)

