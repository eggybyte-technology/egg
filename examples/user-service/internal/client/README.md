# Service-to-Service Communication with clientx

This directory demonstrates the egg framework's recommended pattern for service-to-service communication using `clientx`.

## Overview

The `client` package provides type-safe wrappers for calling other microservices. It showcases the production-ready features of `clientx`:

- **Automatic retry** with exponential backoff
- **Circuit breaker** to prevent cascade failures
- **Internal token injection** for service-to-service authentication
- **Configurable timeouts** for reliability
- **Connection pooling** for performance

## Usage Pattern

### 1. Create Client Wrapper

Create a wrapper struct for each external service:

```go
// greet_client.go
package client

import (
    "time"
    "connectrpc.com/connect"
    "go.eggybyte.com/egg/clientx"
    greetv1connect "go.eggybyte.com/egg/examples/minimal-connect-service/gen/go/greet/v1/greetv1connect"
)

type GreetClient struct {
    client greetv1connect.GreeterServiceClient
}

func NewGreetClient(baseURL, internalToken string) *GreetClient {
    client := clientx.NewConnectClient(
        baseURL,
        "greet-service",
        func(httpClient connect.HTTPClient, url string, opts ...connect.ClientOption) greetv1connect.GreeterServiceClient {
            return greetv1connect.NewGreeterServiceClient(httpClient, url, opts...)
        },
        clientx.WithTimeout(10*time.Second),      // Production timeout
        clientx.WithRetry(3),                     // Retry up to 3 times
        clientx.WithCircuitBreaker(true),          // Enable circuit breaker
        clientx.WithInternalToken(internalToken), // Auto-inject internal token
    )
    
    return &GreetClient{client: client}
}

func (c *GreetClient) Client() greetv1connect.GreeterServiceClient {
    return c.client
}
```

### 2. Initialize Client at Startup

Initialize clients once during service startup in `registerServices`:

```go
// cmd/server/main.go
func registerServices(app *servicex.App, cfg *config.AppConfig) error {
    // ... database initialization ...
    
    // Initialize client for service-to-service communication
    var greetClient *client.GreetClient
    if cfg.GreetServiceURL != "" {
        greetClient = client.NewGreetClient(cfg.GreetServiceURL, app.InternalToken())
        app.Logger().Info("greet service client initialized",
            "url", cfg.GreetServiceURL,
            "has_token", app.InternalToken() != "")
    }
    
    // Pass client to service layer
    userService := service.NewUserService(userRepo, app.Logger(), greetClient)
    // ...
}
```

### 3. Use Client in Service Layer

Use the client in your service layer:

```go
// internal/service/user_service.go
func (s *userService) GetGreeting(ctx context.Context, userName string) (string, error) {
    if s.greetClient == nil {
        return "", errors.New(errors.CodeUnimplemented, "greet service not configured")
    }
    
    req := connect.NewRequest(&greetv1.SayHelloRequest{
        Name:     userName,
        Language: "en",
    })
    
    resp, err := s.greetClient.Client().SayHello(ctx, req)
    if err != nil {
        return "", errors.Wrap(errors.CodeUnavailable, "call greet service", err)
    }
    
    return resp.Msg.Message, nil
}
```

## Configuration

Configure service URLs via environment variables:

```yaml
# docker-compose.services.yaml
environment:
  GREET_SERVICE_URL: "http://minimal-service:8080"
  INTERNAL_TOKEN: "dev-internal-secret-token-12345"
```

Or via `configx.BaseConfig` extension:

```go
type AppConfig struct {
    configx.BaseConfig
    GreetServiceURL string `env:"GREET_SERVICE_URL" default:"http://minimal-service:8080"`
}
```

## Best Practices

1. **Create clients once at startup**: Clients are safe for concurrent use and should be reused
2. **Use internal token**: Always pass `app.InternalToken()` to client constructors for service-to-service auth
3. **Handle client nil gracefully**: Optional clients should be checked for nil before use
4. **Configure timeouts**: Use appropriate timeouts (default: 10s for production)
5. **Enable circuit breaker**: Always enable circuit breaker in production to prevent cascade failures
6. **Error handling**: Wrap errors with appropriate error codes (`CodeUnavailable` for service failures)

## Key Features Demonstrated

- ✅ **clientx.NewConnectClient**: Production-ready client creation
- ✅ **WithInternalToken**: Automatic token injection
- ✅ **WithTimeout**: Configurable timeouts
- ✅ **WithRetry**: Automatic retry with exponential backoff
- ✅ **WithCircuitBreaker**: Circuit breaker pattern
- ✅ **Service layer integration**: Clean separation of concerns

This pattern ensures all microservices follow the same standards for inter-service communication.

