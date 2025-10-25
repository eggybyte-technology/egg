// Package main demonstrates a minimal Connect service using the servicex library.
//
// Overview:
//
//	This example showcases the simplest way to create a production-ready Connect
//	service using the egg framework's servicex aggregator. It implements a greeting
//	service with both unary and streaming RPC methods.
//
// Key Features:
//   - Minimal boilerplate: servicex handles all infrastructure setup
//   - Console logging: human-readable colored logs for development
//   - Automatic health checks: /health endpoint with readiness probes
//   - Automatic metrics: /metrics endpoint with Prometheus format
//   - Graceful shutdown: proper cleanup on termination signals
//   - Multi-language support: greetings in English, Spanish, French, German, Chinese
//
// Architecture:
//
//	This is a single-file service demonstrating the egg framework's philosophy:
//	start simple, scale when needed. For larger services, see user-service example
//	which demonstrates proper layering (handler/service/repository).
//
// Usage:
//
//	Run directly:
//	  go run main.go
//
//	Build and run:
//	  go build -o greet-service
//	  ./greet-service
//
//	Configure via environment:
//	  SERVICE_NAME=greet HTTP_PORT=8080 ./greet-service
//
// Endpoints:
//   - HTTP: 8080 (configurable via HTTP_PORT)
//   - Health: 8081 (configurable via HEALTH_PORT)
//   - Metrics: 9091 (configurable via METRICS_PORT)
//
// Dependencies:
//   - servicex: unified service initialization (L4)
//   - configx: configuration management (L2)
//   - logx: structured logging (L1)
//   - connectx: Connect interceptor stack (L3)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/configx"
	"github.com/eggybyte-technology/egg/core/log"
	greetv1 "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1"
	greetv1connect "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1/greetv1connect"
	"github.com/eggybyte-technology/egg/logx"
	"github.com/eggybyte-technology/egg/servicex"
)

// GreeterService implements the greeting service with multi-language support.
//
// This service demonstrates Connect RPC patterns:
//   - Unary RPC: SayHello (single request -> single response)
//   - Server streaming: SayHelloStream (single request -> multiple responses)
//
// Concurrency:
//
//	Safe for concurrent use. Each method receives its own context and operates
//	independently. The logger is safe for concurrent use.
//
// Error Handling:
//
//	All errors are returned as Connect errors with appropriate codes.
//	Network errors and context cancellation are handled gracefully.
type GreeterService struct {
	logger log.Logger
}

// SayHello handles unary greeting requests with multi-language support.
//
// Parameters:
//   - ctx: request context for cancellation and deadlines
//   - req: greeting request containing name and language preference
//
// Returns:
//   - *connect.Response[greetv1.SayHelloResponse]: greeting message with timestamp
//   - error: nil on success; Connect error on failure
//
// Behavior:
//   - Default name is "World" if not provided
//   - Default language is "en" (English) if not provided
//   - Supported languages: en, es, fr, de, zh
//   - Unsupported languages fall back to English
//   - Logs each request with name and language
//
// Concurrency:
//
//	Safe for concurrent use. Each request is independent.
func (s *GreeterService) SayHello(ctx context.Context, req *connect.Request[greetv1.SayHelloRequest]) (*connect.Response[greetv1.SayHelloResponse], error) {
	name := req.Msg.Name
	if name == "" {
		name = "World"
	}

	// Add language-specific greeting
	language := req.Msg.Language
	if language == "" {
		language = "en"
	}

	var greeting string
	switch language {
	case "es":
		greeting = "Hola"
	case "fr":
		greeting = "Bonjour"
	case "de":
		greeting = "Hallo"
	case "zh":
		greeting = "你好"
	default:
		greeting = "Hello"
	}

	s.logger.Info("greeting request", "name", name, "language", language)

	response := &greetv1.SayHelloResponse{
		Message:   fmt.Sprintf("%s, %s!", greeting, name),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return connect.NewResponse(response), nil
}

// SayHelloStream handles server-streaming greeting requests.
//
// This method demonstrates server-side streaming in Connect: it sends multiple
// greeting messages back to the client over a single connection.
//
// Parameters:
//   - ctx: request context for cancellation and deadlines
//   - req: streaming request containing name and message count
//   - stream: server stream for sending multiple responses
//
// Returns:
//   - error: nil on success; error if stream fails or context is cancelled
//
// Behavior:
//   - Default name is "World" if not provided
//   - Default count is 5 messages if not provided or negative
//   - Sends one message per 100ms (simulating processing time)
//   - Logs the initial request with name and count
//   - Each message includes a sequence number
//   - Honors context cancellation for early termination
//
// Concurrency:
//
//	Safe for concurrent use. Each stream is independent.
//
// Performance:
//
//	Stream rate: ~10 messages per second (100ms interval)
//	Memory: O(1) per stream (no buffering)
func (s *GreeterService) SayHelloStream(ctx context.Context, req *connect.Request[greetv1.SayHelloStreamRequest], stream *connect.ServerStream[greetv1.SayHelloStreamResponse]) error {
	name := req.Msg.Name
	if name == "" {
		name = "World"
	}

	count := req.Msg.Count
	if count <= 0 {
		count = 5
	}

	s.logger.Info("streaming request", "name", name, "count", count)

	for i := int32(1); i <= count; i++ {
		response := &greetv1.SayHelloStreamResponse{
			Message:  fmt.Sprintf("Hello, %s! (Message %d)", name, i),
			Sequence: i,
		}

		if err := stream.Send(response); err != nil {
			return err
		}

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func main() {
	// Create context for the service
	ctx := context.Background()

	// Create console logger for development (human-readable)
	logger := logx.New(
		logx.WithFormat(logx.FormatConsole),
		logx.WithLevel(slog.LevelInfo),
		logx.WithColor(true),
	)

	// Initialize configuration (uses BaseConfig for standard settings)
	cfg := &configx.BaseConfig{}

	// Run the service with servicex
	err := servicex.Run(ctx,
		servicex.WithService("greet-service", "0.1.0"),
		servicex.WithLogger(logger),
		servicex.WithConfig(cfg),
		servicex.WithRegister(registerServices),
	)
	if err != nil {
		logger.Error(err, "service failed to start")
	}
}

// registerServices registers all Connect RPC handlers with the application.
//
// This function is called by servicex during initialization. It demonstrates
// the standard pattern for registering Connect services:
//  1. Create service implementation with dependencies (logger, etc.)
//  2. Wrap with Connect handler using generated code
//  3. Apply interceptors from servicex (logging, metrics, tracing, etc.)
//  4. Register the handler path with the HTTP mux
//
// Parameters:
//   - app: servicex application instance providing logger, mux, and interceptors
//
// Returns:
//   - error: nil on success; error if registration fails
//
// Concurrency:
//
//	Called once during service startup, not safe for concurrent use.
//
// Note:
//
//	The Connect handler path is automatically generated by protoc-gen-connect-go
//	based on the service definition in the .proto file.
func registerServices(app *servicex.App) error {
	// Create greeter service with logger
	greeterService := &GreeterService{
		logger: app.Logger(),
	}

	// Register Connect handler
	path, handler := greetv1connect.NewGreeterServiceHandler(
		greeterService,
		connect.WithInterceptors(app.Interceptors()...),
	)

	app.Mux().Handle(path, handler)

	app.Logger().Info("greet service initialized successfully")
	return nil
}
