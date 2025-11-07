// Package client provides HTTP clients for calling other microservices.
//
// Overview:
//   - Responsibility: Initialize and manage Connect clients for service-to-service communication
//   - Key Types: Client factories that create configured Connect clients
//   - Concurrency Model: Clients are safe for concurrent use
//   - Error Semantics: Client initialization errors are returned immediately
//   - Performance Notes: Clients are created once and reused across requests
//
// Usage:
//
//	greetClient := client.NewGreetClient("http://minimal-service:8080", internalToken)
//	response, err := greetClient.SayHello(ctx, connect.NewRequest(&greetv1.SayHelloRequest{
//	    Name:     "User Service",
//	    Language: "en",
//	}))
//
// This demonstrates the egg framework's recommended pattern for service-to-service
// communication using clientx with production-ready features like retry, circuit breaker,
// and automatic internal token injection.
package client

import (
	"time"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/clientx"
	greetv1connect "go.eggybyte.com/egg/examples/minimal-connect-service/gen/go/greet/v1/greetv1connect"
)

// GreetClient wraps the Connect client for GreeterService.
// This provides a type-safe interface for calling the greet service.
type GreetClient struct {
	client greetv1connect.GreeterServiceClient
}

// NewGreetClient creates a new GreetClient with production-ready configuration.
//
// This function demonstrates the egg framework's clientx usage pattern:
//   - Uses clientx.NewConnectClient for automatic retry and circuit breaker
//   - Automatically injects internal token for service-to-service authentication
//   - Configures appropriate timeouts and retry policies
//   - Creates reusable clients that can be shared across goroutines
//
// Parameters:
//   - baseURL: Base URL of the greet service (e.g., "http://minimal-service:8080")
//   - internalToken: Internal token for service-to-service authentication (from INTERNAL_TOKEN env)
//
// Returns:
//   - *GreetClient: Configured client ready for use
//
// Configuration:
//   - Timeout: 10 seconds (production-ready default)
//   - Max Retries: 3 attempts with exponential backoff
//   - Circuit Breaker: Enabled (prevents cascade failures)
//   - Internal Token: Automatically added to all requests
//
// Concurrency:
//   - Returns a client safe for concurrent use
func NewGreetClient(baseURL, internalToken string) *GreetClient {
	// Use clientx.NewConnectClient for production-ready client with:
	// - Automatic retry with exponential backoff
	// - Circuit breaker to prevent cascade failures
	// - Internal token injection for service-to-service auth
	// - Configurable timeouts
	client := clientx.NewConnectClient(
		baseURL,
		"greet-service",
		func(httpClient connect.HTTPClient, url string, opts ...connect.ClientOption) greetv1connect.GreeterServiceClient {
			return greetv1connect.NewGreeterServiceClient(httpClient, url, opts...)
		},
		clientx.WithTimeout(10*time.Second),      // Production timeout
		clientx.WithRetry(3),                     // Retry up to 3 times
		clientx.WithCircuitBreaker(true),         // Enable circuit breaker
		clientx.WithInternalToken(internalToken), // Auto-inject internal token
	)

	return &GreetClient{
		client: client,
	}
}

// Client returns the underlying Connect client for advanced usage.
func (c *GreetClient) Client() greetv1connect.GreeterServiceClient {
	return c.client
}
