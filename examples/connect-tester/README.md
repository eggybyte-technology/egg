# Connect Service Tester

A comprehensive testing tool for Connect RPC services built with the egg framework.

## Overview

The Connect Tester provides an easy way to verify that your Connect services are working correctly. It supports testing both minimal services (like the greet service) and full CRUD services (like the user service).

## Key Features

- **Multi-service support**: Test minimal services and CRUD services
- **Comprehensive coverage**: Tests unary RPC, server streaming, and all CRUD operations
- **Colored output**: Green for success, red for failure, easy to scan
- **Detailed metrics**: Request timing, success rates, error diagnostics
- **Uses egg libraries**: logx for logging, clientx for resilient HTTP clients
- **Error scenario testing**: Validates error handling for edge cases

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
```

This tests:
- `SayHello`: Unary greeting request
- `SayHelloStream`: Server streaming greeting

### Test User Service (Full CRUD)

```bash
./connect-tester http://localhost:8082 user-service
```

This runs a comprehensive test suite:
- Create a test user
- Get the created user
- Update the user
- List users with pagination
- Delete the user
- Test error scenarios (non-existent user, empty fields, etc.)

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

## Output Format

The tester provides clear, colored output:

```
✓ PASS SayHello: message="Hello, Tester!" duration=45ms
✓ PASS SayHelloStream: messages=3 duration=312ms
✓ PASS CreateUser: user_id="abc-123" duration=67ms
✓ PASS GetUser: user_id="abc-123" email="test@example.com" duration=23ms
✗ FAIL DeleteUser: error="user not found"
```

At the end, a summary is displayed:

```
Test Summary: total=5 passed=4 failed=1 success_rate=80.00%
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
