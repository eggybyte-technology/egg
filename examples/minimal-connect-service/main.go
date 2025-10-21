// Package main demonstrates a minimal Connect service using the servicex library.
//
// This example shows how to:
// - Set up a Connect service with minimal code using servicex
// - Use servicex for unified service initialization
// - Integrate OpenTelemetry for observability
// - Handle graceful shutdown automatically
// - Configure health and metrics endpoints
package main

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/core/identity"
	"github.com/eggybyte-technology/egg/core/log"
	greetv1 "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1"
	greetv1connect "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1/greetv1connect"
	"github.com/eggybyte-technology/egg/servicex"
)

// SimpleLogger is a basic implementation of the log.Logger interface
type SimpleLogger struct{}

func (l *SimpleLogger) With(kv ...any) log.Logger {
	return l // Simple implementation - doesn't store context
}

func (l *SimpleLogger) Debug(msg string, kv ...any) {
	fmt.Printf("[DEBUG] %s %v\n", msg, kv)
}

func (l *SimpleLogger) Info(msg string, kv ...any) {
	fmt.Printf("[INFO] %s %v\n", msg, kv)
}

func (l *SimpleLogger) Warn(msg string, kv ...any) {
	fmt.Printf("[WARN] %s %v\n", msg, kv)
}

func (l *SimpleLogger) Error(err error, msg string, kv ...any) {
	if err != nil {
		fmt.Printf("[ERROR] %s: %v %v\n", msg, err, kv)
	} else {
		fmt.Printf("[ERROR] %s %v\n", msg, kv)
	}
}

// GreeterService implements the greeting service
type GreeterService struct{}

func (s *GreeterService) SayHello(ctx context.Context, req *connect.Request[greetv1.SayHelloRequest]) (*connect.Response[greetv1.SayHelloResponse], error) {
	// Extract user info from context if available
	if user, ok := identity.UserFrom(ctx); ok {
		logger.Info("User greeting request", log.Str("user_id", user.UserID))
	}

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

	response := &greetv1.SayHelloResponse{
		Message:   fmt.Sprintf("%s, %s!", greeting, name),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return connect.NewResponse(response), nil
}

func (s *GreeterService) SayHelloStream(ctx context.Context, req *connect.Request[greetv1.SayHelloStreamRequest], stream *connect.ServerStream[greetv1.SayHelloStreamResponse]) error {
	name := req.Msg.Name
	if name == "" {
		name = "World"
	}

	count := req.Msg.Count
	if count <= 0 {
		count = 5
	}

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

// AppConfig extends BaseConfig with application-specific settings
type AppConfig struct {
	// Application-specific configuration
	SlowRequestMillis int64 `env:"SLOW_REQUEST_MILLIS" default:"1000"`
	RateLimitQPS      int   `env:"RATE_LIMIT_QPS" default:"100"`
	EnableDebugLogs   bool  `env:"ENABLE_DEBUG_LOGS" default:"false"`
}

var logger log.Logger

func main() {
	// Initialize logger
	logger = &SimpleLogger{}
	logger.Info("Starting minimal Connect service")

	// Create context
	ctx := context.Background()

	// Initialize configuration
	var cfg AppConfig

	// Run the service using servicex
	err := servicex.Run(ctx, servicex.Options{
		ServiceName: "greet-service",
		Config:      &cfg,
		Register: func(app *servicex.App) error {
			// Create Connect handler
			greeterService := &GreeterService{}
			path, handler := greetv1connect.NewGreeterServiceHandler(
				greeterService,
				connect.WithInterceptors(app.Interceptors()...),
			)

			// Register handler
			app.Mux().Handle(path, handler)

			logger.Info("Greet service initialized successfully")
			return nil
		},
		EnableTracing:     true,
		EnableHealthCheck: true,
		EnableMetrics:     true,
		EnableDebugLogs:   false,
		SlowRequestMillis: 1000,
		PayloadAccounting: true,
		ShutdownTimeout:   15 * time.Second,
		Logger:            logger,
	})
	if err != nil {
		logger.Error(err, "Service failed")
		return
	}

	logger.Info("Service stopped gracefully")
}
