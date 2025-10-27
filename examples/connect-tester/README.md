# Connect Service Tester

A comprehensive testing tool for Connect RPC services built with the egg framework.

## Overview

The Connect Tester provides an easy way to verify that your Connect services are working correctly. It supports testing both minimal services (like the greet service) and full CRUD services (like the user service).

## Key Features

- **Multi-service support**: Test minimal services and CRUD services
- **Comprehensive coverage**: Tests unary RPC, server streaming, and all CRUD operations
- **Metrics endpoint testing**: Validates Prometheus `/metrics` endpoint availability and format
- **Enhanced test scenarios**: Multi-language greetings, batch operations, error handling
- **Colored output**: Green for success, red for failure, easy to scan
- **Detailed metrics**: Request timing, success rates, error diagnostics
- **Uses egg libraries**: logx for logging, clientx for resilient HTTP clients
- **Error scenario testing**: Validates error handling for edge cases (empty names, non-existent IDs, invalid inputs)

## Installation

Build the tester:

```bash
cd examples/connect-tester
go build -o connect-tester
```

## Usage

### Test Minimal Service (Greet)

```bash
./connect-tester http://localhost:8080 minimal-service

# Or use go run
go run main.go http://localhost:8080 minimal-service
```

This tests:
- `SayHello` with multiple languages (English, Spanish, French, German, Chinese)
- `SayHello` with empty name (error scenario)
- `SayHelloStream` with various counts (1, 3, 5, 10)
- `SayHelloStream` with zero count (default behavior)
- **Metrics endpoint** at `http://localhost:9091/metrics`
  - Validates HTTP 200 response
  - Checks Prometheus format (# HELP, # TYPE markers)
  - Verifies `target_info` metric with service name
  - Counts exported metrics

### Test User Service (Full CRUD)

```bash
./connect-tester http://localhost:8082 user-service

# Or use go run
go run main.go http://localhost:8082 user-service
```

This runs a comprehensive test suite:
- **Batch creation**: Create multiple users (3 users)
- **Get user**: Retrieve user by ID
- **Update user**: Modify user email and name
- **List users**: Pagination with different page sizes
- **Delete user**: Remove user and verify deletion
- **Error scenarios**:
  - Get non-existent user (should return NotFound)
  - Create user with empty email (should return InvalidArgument)
  - Create user with empty name (should return InvalidArgument)
- **Metrics endpoint** at `http://localhost:9092/metrics`
  - Validates HTTP 200 response
  - Checks Prometheus format (# HELP, # TYPE markers)
  - Verifies `target_info` metric with service name
  - Counts exported metrics

### Test Specific User Operations

```bash
# Create a user
./connect-tester http://localhost:8082 user-service create email@test.com "Test User"

# Get a user by ID
./connect-tester http://localhost:8082 user-service get <user-id>

# Update a user
./connect-tester http://localhost:8082 user-service update <user-id> email@test.com "Updated Name"

# Delete a user
./connect-tester http://localhost:8082 user-service delete <user-id>

# List users with pagination
./connect-tester http://localhost:8082 user-service list 1 10
```

## Metrics Endpoint Testing

The tester automatically derives the metrics endpoint URL from the service base URL.

### Port Mapping

The tester uses a **smart port mapping** approach:

1. **Known docker-compose mappings** (defined in `docker-compose.services.yaml`):
   - `http://localhost:8080` → Metrics at `http://localhost:9091/metrics` (minimal-service)
   - `http://localhost:8082` → Metrics at `http://localhost:9092/metrics` (user-service)

2. **Fallback rule** for unknown ports: `Metrics Port = HTTP Port + 1011`
   - `http://example.com:8000` → Metrics at `http://example.com:9011/metrics`
   - `http://localhost:3000` → Metrics at `http://localhost:4011/metrics`

### Override Metrics URL

You can override the automatic derivation by setting the `METRICS_URL` environment variable:

```bash
# Test with custom metrics endpoint
METRICS_URL=http://custom-host:9091/metrics ./connect-tester http://localhost:8080 minimal-service

# Test remote service with local metrics endpoint
METRICS_URL=http://localhost:9091/metrics ./connect-tester http://remote-host:8080 minimal-service
```

This is useful when:
- Services are running in Docker with port mapping
- Testing services behind a proxy

## Metrics Validation

The tester performs comprehensive metrics validation:

### Basic Validation
- **HTTP 200 status** - Metrics endpoint is accessible
- **Prometheus format** - Response contains valid Prometheus text format
- **Target info** - Service metadata is present

### RPC Metrics Validation
The tester verifies that RPC metrics are automatically collected by the connectx metrics interceptor:

| Metric Name | Type | Labels | Verification |
| --- | --- | --- | --- |
| `rpc_requests_total` | Counter | `rpc_service`, `rpc_method`, `rpc_code` | Metric exists in output |
| `rpc_request_duration_seconds` | Histogram | `rpc_service`, `rpc_method`, `rpc_code` | Metric exists in output |

### Request Count Validation
The tester counts successful RPC calls made during tests and verifies that:
- `rpc_requests_total{rpc_code="ok"}` >= expected call count
- Ensures the metrics interceptor is working correctly
- Validates that labels follow the whitelist: `rpc_service`, `rpc_method`, `rpc_code`

**Example test output:**
```
✓ PASS Metrics_Endpoint duration=45ms metrics=127 service=greet-service rpc_requests=11
✓ PASS Metrics_RPC_rpc_requests_total - metric exists
✓ PASS Metrics_RPC_rpc_request_duration_seconds - metric exists
✓ PASS Metrics_RPC_Count actual=11 expected_min=11
```

**Verified naming conventions:**
- Counter suffix: `*_total`
- Duration suffix: `*_seconds` (measured in seconds, not milliseconds)
- Size suffix: `*_bytes` (measured in bytes)
- Histogram buckets: Standard buckets for duration and size

### Metric Parsing
The tester includes a Prometheus text format parser that:
- Extracts metric names, labels, and values
- Supports filtering metrics by label
- Aggregates values across label dimensions

This enables precise validation of metric values, not just presence.
- Metrics are exposed on a different host than the service

## Output Format

The tester provides clear, colored output:

```
✓ PASS SayHello_en: message="Hello, World!" duration=45ms
✓ PASS SayHelloStream_5: messages=5 expected=5 duration=312ms
✓ PASS CreateUser_1: user_id="abc-123" duration=67ms
✓ PASS GetUser: user_id="abc-123" email="test@example.com" duration=23ms
✓ PASS Metrics_Endpoint: duration=15ms metrics=3 service=greet-service
✗ FAIL DeleteUser: error="user not found"
```

At the end, a summary is displayed:

```
Test Summary: total=12 passed=11 failed=1 success_rate=91.67%
```

## Exit Codes

- `0`: All tests passed
- `1`: One or more tests failed

## Dependencies

- `logx`: Structured logging with colors (L1)
- `clientx`: HTTP client with retry and circuit breaker (L3)
- `core/log`: Standardized log interface (L0)

## Example Test Patterns

### Testing with Docker Services

```bash
# Start services
make services-up

# Test minimal service
./connect-tester http://localhost:8080 minimal-service

# Test user service
./connect-tester http://localhost:8082 user-service

# Stop services
make services-down
```

### Integration with CI/CD

```bash
#!/bin/bash
set -e

# Start infrastructure
make infra-up

# Build and start services
make docker-all
make services-up

# Wait for services to be ready
sleep 10

# Run tests
./connect-tester http://localhost:8080 minimal-service
./connect-tester http://localhost:8082 user-service

# Cleanup
make deploy-down
```

## Troubleshooting

### Connection Refused

If you see connection errors:

1. Verify the service is running: `docker ps`
2. Check the port: `curl http://localhost:8080/health`
3. Ensure the URL is correct (no trailing slash)

### Timeout Errors

If requests timeout:

1. Check service logs: `docker logs egg-minimal-service`
2. Verify database connectivity for user-service
3. Increase timeout in main.go if needed

### Test Failures

If tests fail:

1. Check service health endpoints
2. Review service logs for errors
3. Verify database schema is migrated
4. Check environment variables

## License

This package is part of the EggyByte framework and is licensed under the MIT License.
See the root LICENSE file for details.
