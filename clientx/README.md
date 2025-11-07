# egg/clientx

## Overview

`clientx` provides Connect HTTP client factory with production-ready features
including retry logic, circuit breaker, timeouts, and idempotency support.
It simplifies creating resilient Connect-RPC clients.

## Key Features

- Automatic retry with exponential backoff
- Circuit breaker to prevent cascade failures
- Configurable request timeouts
- Idempotency key support
- Connection pooling
- Clean transport abstraction

## Dependencies

Layer: **L3 (Runtime Communication Layer)**  
Depends on: `connectrpc.com/connect`, `github.com/sony/gobreaker`

## Installation

```bash
go get go.eggybyte.com/egg/clientx@latest
```

## Basic Usage

```go
import (
    "go.eggybyte.com/egg/clientx"
    userv1connect "myapp/gen/go/user/v1/userv1connect"
)

func main() {
    // Create HTTP client with resilience features
    httpClient := clientx.NewHTTPClient("https://api.example.com",
        clientx.WithTimeout(5*time.Second),
        clientx.WithRetry(3),
        clientx.WithCircuitBreaker(true),
    )
    
    // Create Connect client
    client := userv1connect.NewUserServiceClient(
        httpClient,
        "https://api.example.com",
    )
    
    // Make requests
    resp, err := client.GetUser(ctx, connect.NewRequest(&userv1.GetUserRequest{
        UserId: "u-123",
    }))
}
```

## Service-to-Service Communication

For service-to-service authentication, use `NewConnectClient()` with `WithInternalToken()`:

```go
import (
    "go.eggybyte.com/egg/clientx"
    greetv1connect "myapp/gen/go/greet/v1/greetv1connect"
)

func createGreetClient(baseURL, internalToken string) greetv1connect.GreeterServiceClient {
    return clientx.NewConnectClient(
        baseURL,
        "greet-service",
        func(httpClient connect.HTTPClient, url string, opts ...connect.ClientOption) greetv1connect.GreeterServiceClient {
            return greetv1connect.NewGreeterServiceClient(httpClient, url, opts...)
        },
        clientx.WithTimeout(10*time.Second),
        clientx.WithRetry(3),
        clientx.WithCircuitBreaker(true),
        clientx.WithInternalToken(internalToken), // Automatically adds X-Internal-Token header
    )
}
```

The internal token is automatically added to all outgoing requests via the `X-Internal-Token` header (configurable via `WithInternalTokenHeader()`).

## Configuration Options

| Option                      | Type           | Description                                |
| --------------------------- | -------------- | ------------------------------------------ |
| `WithTimeout(d)`            | `time.Duration`| Request timeout (default: 30s)             |
| `WithRetry(n)`              | `int`          | Maximum retry attempts (default: 3)        |
| `WithCircuitBreaker(bool)`  | `bool`         | Enable circuit breaker (default: true)     |
| `WithIdempotencyKey(key)`   | `string`       | Custom idempotency header name             |
| `WithInternalToken(token)`  | `string`       | Internal service token (auto-added to requests) |
| `WithInternalTokenHeader(header)` | `string` | Custom header name for internal token (default: `X-Internal-Token`) |

## API Reference

### Options

```go
type Options struct {
    Timeout            time.Duration // Request timeout
    MaxRetries         int           // Maximum retry attempts
    RetryBackoff       time.Duration // Initial backoff duration
    EnableCircuit      bool          // Enable circuit breaker
    CircuitThreshold   uint32        // Circuit breaker failure threshold
    IdempotencyKey     string        // Idempotency key header name
    InternalToken      string        // Internal service token
    InternalTokenHeader string       // Header name for internal token
}
```

### Functions

```go
// NewHTTPClient creates a new HTTP client with Connect interceptors
func NewHTTPClient(baseURL string, opts ...Option) *http.Client

// NewConnectClient creates a Connect client with interceptors
func NewConnectClient[T any](
    baseURL, serviceName string,
    newClient func(connect.HTTPClient, string, ...connect.ClientOption) T,
    opts ...Option,
) T
```

## Architecture

The clientx module provides resilient HTTP transport:

```
clientx/
├── clientx.go           # Public API (~114 lines)
│   ├── Options          # Configuration
│   ├── NewHTTPClient()  # HTTP client factory
│   └── NewConnectClient()  # Connect client helper
└── internal/
    └── retry.go         # Retry transport implementation
        ├── RoundTrip()      # HTTP transport with retry
        ├── shouldRetry()    # Retry decision logic
        └── backoff()        # Exponential backoff
```

**Design Highlights:**
- Wraps standard `http.Transport` with retry logic
- Circuit breaker prevents repeated failures
- Configurable backoff strategy
- Idempotent request detection

## Example: Basic Client

```go
func createUserClient() userv1connect.UserServiceClient {
    httpClient := clientx.NewHTTPClient("https://api.example.com",
        clientx.WithTimeout(10*time.Second),
        clientx.WithRetry(3),
    )
    
    return userv1connect.NewUserServiceClient(
        httpClient,
        "https://api.example.com",
    )
}

func main() {
    client := createUserClient()
    
    resp, err := client.GetUser(context.Background(), connect.NewRequest(&userv1.GetUserRequest{
        UserId: "u-123",
    }))
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("User: %s\n", resp.Msg.User.Name)
}
```

## Example: With Circuit Breaker

```go
// Circuit breaker opens after 5 consecutive failures
httpClient := clientx.NewHTTPClient("https://api.example.com",
    clientx.WithTimeout(5*time.Second),
    clientx.WithRetry(3),
    clientx.WithCircuitBreaker(true),
)

client := userv1connect.NewUserServiceClient(httpClient, "https://api.example.com")

// Make requests
for i := 0; i < 10; i++ {
    resp, err := client.GetUser(ctx, connect.NewRequest(&userv1.GetUserRequest{
        UserId: fmt.Sprintf("u-%d", i),
    }))
    
    if err != nil {
        if errors.Is(err, gobreaker.ErrOpenState) {
            log.Println("Circuit breaker is open, skipping requests")
            time.Sleep(60 * time.Second)  // Wait for circuit to close
            continue
        }
        log.Printf("Request failed: %v\n", err)
        continue
    }
    
    log.Printf("User: %s\n", resp.Msg.User.Name)
}
```

## Example: Custom Retry Strategy

```go
// Configure aggressive retry for critical operations
httpClient := clientx.NewHTTPClient("https://api.example.com",
    clientx.WithTimeout(30*time.Second),
    clientx.WithRetry(5),  // More retries
)

client := paymentv1connect.NewPaymentServiceClient(httpClient, "https://api.example.com")

// Make payment request (will retry up to 5 times on transient failures)
resp, err := client.ProcessPayment(ctx, connect.NewRequest(&paymentv1.ProcessPaymentRequest{
    Amount:   10000,
    Currency: "USD",
}))
```

## Example: Disable Circuit Breaker

```go
// For internal services where circuit breaker isn't needed
httpClient := clientx.NewHTTPClient("http://internal-service:8080",
    clientx.WithTimeout(10*time.Second),
    clientx.WithRetry(2),
    clientx.WithCircuitBreaker(false),  // Disable circuit breaker
)
```

## Retry Logic

### Retryable Conditions

Requests are retried when:
1. Network errors (connection refused, timeout, etc.)
2. HTTP 5xx server errors
3. HTTP 429 (rate limit) errors
4. Transient Connect errors

### Non-Retryable Conditions

Requests are NOT retried for:
1. HTTP 4xx client errors (except 429)
2. Successful responses (2xx)
3. Non-idempotent methods (POST without idempotency key)

### Backoff Strategy

Exponential backoff with jitter:
```
Attempt 1: 100ms
Attempt 2: 200ms
Attempt 3: 400ms
Attempt 4: 800ms
...
```

## Circuit Breaker

### States

1. **Closed** (Normal)
   - Requests pass through
   - Failures are counted

2. **Open** (Failing)
   - Requests are immediately rejected
   - After timeout, transitions to Half-Open

3. **Half-Open** (Testing)
   - Limited requests allowed
   - Success → Closed
   - Failure → Open

### Configuration

```go
// Circuit opens after 5 consecutive failures
CircuitThreshold: 5

// Circuit stays open for 60 seconds before testing
Timeout: 60 * time.Second

// In Half-Open state, allow 3 test requests
MaxRequests: 3
```

## Idempotency Support

```go
// Configure custom idempotency header
httpClient := clientx.NewHTTPClient("https://api.example.com",
    clientx.WithIdempotencyKey("X-Idempotency-Key"),
)

// Client automatically adds idempotency key for POST requests
// Header: X-Idempotency-Key: {generated-uuid}
```

## Connection Pooling

The HTTP client uses Go's default connection pooling:

```go
// Default pool settings (configurable via http.Transport):
MaxIdleConns:        100
MaxIdleConnsPerHost: 2
IdleConnTimeout:     90 * time.Second
```

For custom pooling:
```go
transport := &http.Transport{
    MaxIdleConns:        200,
    MaxIdleConnsPerHost: 10,
    IdleConnTimeout:     120 * time.Second,
}

// Wrap with retry logic
retryTransport := internal.NewRetryTransport(transport, 3, 100*time.Millisecond, nil)

httpClient := &http.Client{
    Timeout:   30 * time.Second,
    Transport: retryTransport,
}
```

## Testing

Mock clients for testing:

```go
func TestUserService(t *testing.T) {
    // Create test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(&userv1.GetUserResponse{
            User: &userv1.User{Id: "u-123", Name: "Test User"},
        })
    }))
    defer server.Close()
    
    // Create client pointing to test server
    httpClient := clientx.NewHTTPClient(server.URL,
        clientx.WithTimeout(1*time.Second),
        clientx.WithRetry(1),
    )
    
    client := userv1connect.NewUserServiceClient(httpClient, server.URL)
    
    // Test
    resp, err := client.GetUser(context.Background(), connect.NewRequest(&userv1.GetUserRequest{
        UserId: "u-123",
    }))
    
    require.NoError(t, err)
    assert.Equal(t, "Test User", resp.Msg.User.Name)
}
```

## Best Practices

1. **Set reasonable timeouts** - Prevent hanging requests
2. **Use circuit breaker for external services** - Protect against cascading failures
3. **Configure appropriate retry counts** - Balance reliability vs latency
4. **Enable idempotency for mutations** - Safe retries for POST/PUT/DELETE
5. **Monitor circuit breaker state** - Alert when circuits open frequently
6. **Test failure scenarios** - Ensure retry logic works as expected

## Performance Considerations

- **Retry Overhead**: Each retry adds latency (100ms base + exponential backoff)
- **Circuit Breaker Overhead**: Minimal (~microseconds per request)
- **Connection Pooling**: Reuse connections for better performance
- **Timeout Configuration**: Balance between resilience and latency

## Stability

**Status**: Stable  
**Layer**: L3 (Runtime Communication)  
**API Guarantees**: Backward-compatible changes only

The clientx module is production-ready and follows semantic versioning.

## License

This package is part of the egg framework and is licensed under the MIT License.
See the root LICENSE file for details.
