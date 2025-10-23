# User Service Example

## Overview

This example demonstrates a complete CRUD microservice using Connect RPC, built with the `servicex` library from the egg framework. It showcases a production-ready service with proper layering (handler → service → repository), optional database integration, and comprehensive observability.

## Features

- **Complete CRUD Operations**: Create, Read, Update, Delete, and List users
- **Layered Architecture**: Clean separation of concerns (handler, service, repository, model)
- **Connect Protocol**: HTTP/2-based RPC with efficient serialization
- **Optional Database**: Supports GORM with MySQL/PostgreSQL, or in-memory mock
- **Observability**: Integrated OpenTelemetry tracing, Prometheus metrics, and structured logging
- **Health Checks**: Built-in health and readiness endpoints
- **Graceful Shutdown**: Automatic signal handling and resource cleanup
- **Production Ready**: Error handling, validation, and context propagation

## Project Structure

```
user-service/
├── api/                        # Protocol Buffer definitions
│   ├── buf.gen.yaml           # Buf code generation config
│   ├── buf.yaml               # Buf module config
│   └── user/
│       └── v1/
│           └── user.proto     # UserService definition
├── cmd/
│   └── server/
│       └── main.go            # Service entrypoint
├── gen/                        # Generated code (by buf)
│   └── go/
│       └── user/
│           └── v1/
│               ├── user.pb.go
│               └── userv1connect/
│                   └── user.connect.go
├── internal/                   # Internal packages
│   ├── config/
│   │   └── app_config.go      # Application configuration
│   ├── handler/
│   │   └── user_handler.go    # Connect protocol handlers
│   ├── model/
│   │   ├── user.go            # User model and GORM annotations
│   │   └── errors.go          # Domain errors
│   ├── repository/
│   │   └── user_repository.go # Data access layer
│   └── service/
│       └── user_service.go    # Business logic layer
├── go.mod                      # Go module definition
├── go.sum                      # Go module checksums
├── Makefile                    # Build and development tasks
├── .env.example                # Example environment variables
└── README.md                   # This file
```

## Prerequisites

- Go 1.23 or later
- Buf CLI (for regenerating protobuf code)
- MySQL or PostgreSQL (optional, defaults to in-memory mock)

## Installation

1. Clone the repository:

```bash
git clone https://github.com/eggybyte-technology/egg.git
cd egg/examples/user-service
```

2. Install dependencies:

```bash
go mod download
```

## Configuration

The service can be configured via environment variables. Copy `.env.example` to `.env` and adjust as needed:

```bash
cp .env.example .env
```

### Environment Variables

#### Service Configuration

| Variable              | Default | Description                           |
| --------------------- | ------- | ------------------------------------- |
| `SERVICE_NAME`        | `user-service` | Service name for observability |
| `SERVICE_VERSION`     | `0.1.0` | Service version                      |
| `HTTP_ADDR`           | `:8080` | HTTP server address                  |
| `HEALTH_ADDR`         | `:8081` | Health check endpoint address        |
| `METRICS_ADDR`        | `:9091` | Metrics endpoint address             |
| `ENABLE_TRACING`      | `true`  | Enable OpenTelemetry tracing         |
| `ENABLE_HEALTH_CHECK` | `true`  | Enable health check endpoint         |
| `ENABLE_METRICS`      | `true`  | Enable Prometheus metrics            |
| `ENABLE_DEBUG_LOGS`   | `false` | Enable debug-level logging           |
| `SLOW_REQUEST_MILLIS` | `1000`  | Threshold for slow request logging   |
| `PAYLOAD_ACCOUNTING`  | `true`  | Enable payload size tracking         |
| `SHUTDOWN_TIMEOUT`    | `15s`   | Graceful shutdown timeout            |

#### Database Configuration (Optional)

| Variable              | Default | Description                           |
| --------------------- | ------- | ------------------------------------- |
| `DB_DRIVER`           | `mysql` | Database driver (mysql, postgres)    |
| `DB_DSN`              | (empty) | Database connection string           |
| `DB_MAX_IDLE`         | `10`    | Maximum idle connections             |
| `DB_MAX_OPEN`         | `100`   | Maximum open connections             |
| `DB_MAX_LIFETIME`     | `1h`    | Maximum connection lifetime          |

Example DSN strings:

```bash
# MySQL
DB_DSN="user:password@tcp(localhost:3306)/userdb?parseTime=true&charset=utf8mb4"

# PostgreSQL
DB_DSN="host=localhost port=5432 user=postgres password=password dbname=userdb sslmode=disable"
```

## Running the Service

### Without Database (In-Memory Mock)

```bash
go run cmd/server/main.go
```

### With Database

1. Set up a database:

```bash
# MySQL
mysql -u root -p -e "CREATE DATABASE userdb;"

# PostgreSQL
psql -U postgres -c "CREATE DATABASE userdb;"
```

2. Configure database connection in `.env`

3. Uncomment database configuration in `cmd/server/main.go`:

```go
Database: &servicex.DatabaseConfig{
    Driver:      "mysql",
    DSN:         cfg.DatabaseDSN,
    MaxIdle:     10,
    MaxOpen:     100,
    MaxLifetime: 1 * time.Hour,
},
Migrate: func(db *gorm.DB) error {
    return db.AutoMigrate(&model.User{})
},
```

4. Run the service:

```bash
go run cmd/server/main.go
```

### Using Make

```bash
make run
```

### Using Docker

```bash
make docker-build
make docker-run
```

## Testing the Service

### Create a User

```bash
curl -X POST http://localhost:8080/user.v1.UserService/CreateUser \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com"}'
```

Expected response:

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "john@example.com",
    "name": "John Doe",
    "created_at": 1698067200,
    "updated_at": 1698067200
  }
}
```

### Get a User

```bash
curl -X POST http://localhost:8080/user.v1.UserService/GetUser \
  -H "Content-Type: application/json" \
  -d '{"id":"550e8400-e29b-41d4-a716-446655440000"}'
```

### Update a User

```bash
curl -X POST http://localhost:8080/user.v1.UserService/UpdateUser \
  -H "Content-Type: application/json" \
  -d '{"id":"550e8400-e29b-41d4-a716-446655440000","name":"Jane Doe","email":"jane@example.com"}'
```

### List Users

```bash
curl -X POST http://localhost:8080/user.v1.UserService/ListUsers \
  -H "Content-Type: application/json" \
  -d '{"page":1,"page_size":10}'
```

### Delete a User

```bash
curl -X POST http://localhost:8080/user.v1.UserService/DeleteUser \
  -H "Content-Type: application/json" \
  -d '{"id":"550e8400-e29b-41d4-a716-446655440000"}'
```

## Observability

### Health Check

```bash
curl http://localhost:8081/health
```

Response:

```json
{
  "status": "ok",
  "timestamp": "2025-10-23T12:34:56Z"
}
```

### Metrics

```bash
curl http://localhost:9091/metrics
```

Key metrics:

- `rpc_server_requests_total`: Total number of RPC requests by method and status
- `rpc_server_request_duration_seconds`: RPC request duration histogram
- `rpc_server_request_size_bytes`: RPC request size histogram
- `rpc_server_response_size_bytes`: RPC response size histogram
- `db_connection_pool_idle`: Current idle database connections
- `db_connection_pool_open`: Current open database connections

### Tracing

Configure OpenTelemetry collector endpoint via environment variables:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

## Architecture

### Layered Design

The service follows a clean layered architecture:

```
┌─────────────────────────────────┐
│     Transport Layer (Handler)   │  ← Connect protocol, request/response mapping
├─────────────────────────────────┤
│     Business Layer (Service)    │  ← Business logic, validation, orchestration
├─────────────────────────────────┤
│     Data Layer (Repository)     │  ← Database operations, queries
├─────────────────────────────────┤
│     Model Layer                 │  ← Data models, GORM annotations
└─────────────────────────────────┘
```

### Request Flow

1. **Handler Layer** (`internal/handler/user_handler.go`):
   - Receives Connect RPC requests
   - Extracts request parameters
   - Delegates to business service
   - Maps domain errors to Connect errors
   - Returns Connect responses

2. **Service Layer** (`internal/service/user_service.go`):
   - Implements business logic
   - Performs validation
   - Orchestrates repository calls
   - Handles domain errors
   - Returns structured responses

3. **Repository Layer** (`internal/repository/user_repository.go`):
   - Executes database queries
   - Handles GORM operations
   - Maps database errors to domain errors
   - Returns data models

4. **Model Layer** (`internal/model/user.go`):
   - Defines data structures
   - GORM annotations for schema
   - Validation logic
   - Domain errors

## API Reference

### UserService

#### CreateUser

Creates a new user with email and name.

**Request:**

```protobuf
message CreateUserRequest {
  string email = 1;  // User email (required, unique)
  string name = 2;   // User name (required)
}
```

**Response:**

```protobuf
message CreateUserResponse {
  User user = 1;     // Created user with generated ID
}
```

**Errors:**

- `INVALID_ARGUMENT`: Email or name is empty
- `ALREADY_EXISTS`: Email already registered
- `INTERNAL`: Database error

#### GetUser

Retrieves a user by ID.

**Request:**

```protobuf
message GetUserRequest {
  string id = 1;     // User ID (required)
}
```

**Response:**

```protobuf
message GetUserResponse {
  User user = 1;     // User details
}
```

**Errors:**

- `INVALID_ARGUMENT`: ID is empty
- `NOT_FOUND`: User not found
- `INTERNAL`: Database error

#### UpdateUser

Updates an existing user's email and/or name.

**Request:**

```protobuf
message UpdateUserRequest {
  string id = 1;     // User ID (required)
  string email = 2;  // New email (required)
  string name = 3;   // New name (required)
}
```

**Response:**

```protobuf
message UpdateUserResponse {
  User user = 1;     // Updated user
}
```

**Errors:**

- `INVALID_ARGUMENT`: ID, email, or name is empty
- `NOT_FOUND`: User not found
- `ALREADY_EXISTS`: New email already registered
- `INTERNAL`: Database error

#### DeleteUser

Deletes a user by ID.

**Request:**

```protobuf
message DeleteUserRequest {
  string id = 1;     // User ID (required)
}
```

**Response:**

```protobuf
message DeleteUserResponse {
  bool success = 1;  // True if deleted successfully
}
```

**Errors:**

- `INVALID_ARGUMENT`: ID is empty
- `NOT_FOUND`: User not found
- `INTERNAL`: Database error

#### ListUsers

Lists users with pagination.

**Request:**

```protobuf
message ListUsersRequest {
  int32 page = 1;      // Page number (default: 1)
  int32 page_size = 2; // Page size (default: 10, max: 100)
}
```

**Response:**

```protobuf
message ListUsersResponse {
  repeated User users = 1;  // List of users
  int32 total = 2;          // Total user count
  int32 page = 3;           // Current page
  int32 page_size = 4;      // Page size used
}
```

**Errors:**

- `INTERNAL`: Database error

## Development

### Regenerating Protocol Buffers

```bash
make generate
```

Or manually:

```bash
cd api && buf generate
```

### Running Tests

```bash
make test
```

### Linting

```bash
make lint
```

### Building

```bash
make build
```

### Database Migrations

The service uses GORM's AutoMigrate for schema management. To add new fields:

1. Update the `User` model in `internal/model/user.go`
2. Run the service (AutoMigrate will update the schema)

For production, consider using a proper migration tool like [golang-migrate](https://github.com/golang-migrate/migrate).

## Key Concepts

### servicex Integration

This example uses `servicex.Run()` for unified service initialization:

```go
err := servicex.Run(ctx, servicex.Options{
    ServiceName: "user-service",
    Config:      &cfg,
    Database: &servicex.DatabaseConfig{
        Driver:      "mysql",
        DSN:         cfg.DatabaseDSN,
        MaxIdle:     10,
        MaxOpen:     100,
        MaxLifetime: 1 * time.Hour,
    },
    Migrate: func(db *gorm.DB) error {
        return db.AutoMigrate(&model.User{})
    },
    Register: func(app *servicex.App) error {
        // Initialize layers
        userRepo := repository.NewUserRepository(app.DB())
        userService := service.NewUserService(userRepo, app.Logger())
        userHandler := handler.NewUserHandler(userService, app.Logger())
        
        // Register Connect handler
        path, connectHandler := userv1connect.NewUserServiceHandler(
            userHandler,
            connect.WithInterceptors(app.Interceptors()...),
        )
        app.Mux().Handle(path, connectHandler)
        return nil
    },
})
```

### Error Handling

The service uses structured errors from `core/errors`:

```go
// Define domain errors
var (
    ErrUserNotFound = errors.New(errors.CodeNotFound, "user not found")
    ErrEmailExists  = errors.New(errors.CodeAlreadyExists, "email already exists")
)

// Wrap errors with context
return nil, errors.Wrap(errors.CodeInternal, "create user", err)
```

### Logging

The service uses structured logging from `core/log`:

```go
logger.Info("User created successfully",
    log.Str("user_id", user.ID),
    log.Str("email", user.Email))
```

## Troubleshooting

### Service doesn't start

1. Check if ports are already in use:

```bash
lsof -i :8080
lsof -i :8081
lsof -i :9091
```

2. Check database connection if configured
3. Check logs for initialization errors

### Database connection errors

1. Verify database credentials in `.env`
2. Ensure database server is running
3. Test connection manually:

```bash
# MySQL
mysql -h localhost -u user -p dbname

# PostgreSQL
psql -h localhost -U user dbname
```

### "email already exists" errors

The service enforces unique email constraints. Use a different email or delete the existing user first.

### Slow requests

Check the `SLOW_REQUEST_MILLIS` threshold and adjust as needed. Slow requests are automatically logged with full context.

## License

This example is part of the EggyByte egg framework and is licensed under the MIT License. See the root LICENSE file for details.

## Related Documentation

- [egg Framework Documentation](../../docs/)
- [servicex Module](../../servicex/README.md)
- [connectx Module](../../connectx/README.md)
- [core/errors Module](../../core/errors/README.md)
- [core/log Module](../../core/log/README.md)
- [Connect Protocol](https://connectrpc.com/)
- [GORM Documentation](https://gorm.io/)

