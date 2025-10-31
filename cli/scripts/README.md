# Test Scripts Documentation

This directory contains modular test scripts for the Egg CLI integration tests.

## File Structure

```
cli/scripts/
├── test-cli.sh          # Main test orchestrator (entry point)
├── test-config.sh       # Test configuration and constants
├── test-helpers.sh      # Common helper functions
└── test-compose.sh      # Docker Compose service testing (health checks + RPC)
```

## Module Descriptions

### test-cli.sh
Main test orchestrator that coordinates all test modules. This is the entry point for running integration tests.

**Responsibilities:**
- Test orchestration and sequencing
- Project initialization tests
- Service creation tests
- Build tests
- Calls modular test functions

**Usage:**
```bash
./cli/scripts/test-cli.sh [--remove]
```

### test-config.sh
Defines test configuration constants and variables used across all test scripts.

**Contains:**
- Test project configuration (name, paths, services)
- Service port definitions
- Connect RPC path definitions
- CLI binary path
- Command-line argument parsing

**Dependencies:**
- Sources `logger.sh` from project root
- Sources `test-helpers.sh`

### test-helpers.sh
Provides common helper functions used across all test scripts.

**Key Functions:**
- `run_egg_command()` - Execute egg CLI commands with formatted output
- `check_file()` / `check_dir()` - File/directory validation
- `check_file_content()` - Content validation
- `wait_for_endpoint()` - Wait for HTTP endpoint with retry
- `wait_for_endpoint_pattern()` - Wait for endpoint with pattern matching
- `call_connect_rpc()` - Call Connect RPC endpoints

**Dependencies:**
- Requires `logger.sh` to be sourced first

### test-compose.sh
Tests Docker Compose services including health checks and RPC endpoints.

**Key Functions:**
- `test_compose_services()` - Main test function for compose services
- `test_ping_service_rpc()` - Test Ping service RPC endpoint
- `test_user_service_crud()` - Test User service CRUD operations
- `wait_for_endpoint_docker()` - Wait for endpoint using Docker internal network
- `docker_compose_curl()` - Execute curl commands inside Docker network

**Test Flow:**
1. Start Docker Compose services
2. Wait for health checks to pass (using Docker service names)
3. Test metrics endpoints (with pattern matching)
4. Test RPC endpoints (Ping, CRUD) via Docker network
5. Services remain running after tests for manual inspection

**Docker Network Testing:**
- All tests access services via Docker internal network DNS
- Uses `docker compose exec` to run wget commands inside containers
- Service URLs: `http://service-name:port/path` (e.g., `http://user:8080/health`)
- No localhost port mappings required

**Dependencies:**
- Requires `test-config.sh` (which sources `test-helpers.sh`)

## Usage Examples

### Run Full Test Suite
```bash
cd /Users/fengguangyao/eggybyte/projects/go/egg
./cli/scripts/test-cli.sh
```

### Run Tests and Clean Up
```bash
./cli/scripts/test-cli.sh --remove
```

### Test Only Compose Services (after services are built)

**Option 1: Run as standalone script (recommended)**
```bash
# From project root
./cli/scripts/test-compose.sh

# With custom test directory
./cli/scripts/test-compose.sh --test-dir /path/to/project

# Show help
./cli/scripts/test-compose.sh --help
```

**Option 2: Source and call function**
```bash
source cli/scripts/test-config.sh
source cli/scripts/test-compose.sh
cd cli/tmp/test-project/deploy/compose
API_SUCCESS=true
test_compose_services

# Services remain running after tests
# To stop manually: docker compose down
```

## RPC Testing

The test suite includes comprehensive RPC endpoint testing via Docker internal network:

### Network Access
- All tests execute inside Docker containers using `docker compose exec`
- Services accessed via Docker DNS: `http://service-name:port/path`
- No localhost port mappings required
- Tests use executor service (first backend service) to run wget commands

### Ping Service
- **Endpoint**: `http://ping:8090/eggybyte_test.test_project.ping.v1.PingService/Ping`
- **Method**: POST
- **Request**: `{"message": "Hello from test"}`
- **Validation**: Checks for "message" in response

### User Service (CRUD)
- **Base URL**: `http://user:8080`
- **Create**: Creates a user with email and name
- **Get**: Retrieves user by ID
- **Update**: Updates user email and name
- **List**: Lists users with pagination
- **Delete**: Deletes user by ID

All CRUD operations are tested sequentially, with the created user ID used for subsequent operations.

## Architecture Benefits

### Modularity
- Each module has a single, clear responsibility
- Easy to test individual components
- Functions can be reused across different test scenarios

### Maintainability
- Changes to health checks only affect `test-compose.sh`
- Configuration changes only affect `test-config.sh`
- Helper function improvements benefit all tests

### Extensibility
- Easy to add new test modules (e.g., `test-k8s.sh`)
- Easy to add new RPC tests to `test-compose.sh`
- Easy to add new helper functions to `test-helpers.sh`

## Dependencies

All scripts depend on:
- `scripts/logger.sh` - Unified logging from project root
- `wget` - For HTTP endpoint testing (executed inside Docker containers, wget is available in containers)
- `docker` / `docker compose` - For container testing and network access
- `egg` CLI binary - Must be built before running tests
- `jq` (optional) - For JSON parsing in frontend service detection

## Error Handling

- Scripts use `set -e` for immediate failure on errors
- Error trap in `test-cli.sh` ensures cleanup on failure
- RPC tests return non-zero exit codes on failure
- Health check failures prevent RPC tests from running

## Future Enhancements

Potential additions:
- `test-k8s.sh` - Kubernetes deployment testing
- `test-performance.sh` - Performance and load testing
- `test-security.sh` - Security scanning tests
- More granular RPC test functions for other services

