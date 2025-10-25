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
	"github.com/eggybyte-technology/egg/configx"
	greetv1 "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1"
	greetv1connect "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1/greetv1connect"
	"github.com/eggybyte-technology/egg/servicex"
)

// GreeterService implements the greeting service
type GreeterService struct{}

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
	// Inherit BaseConfig for standard configuration (ports, service info, etc.)
	configx.BaseConfig `env:""`

	// Application-specific configuration
	SlowRequestMillis int64 `env:"SLOW_REQUEST_MILLIS" default:"1000"`
	RateLimitQPS      int   `env:"RATE_LIMIT_QPS" default:"100"`
	EnableDebugLogs   bool  `env:"ENABLE_DEBUG_LOGS" default:"false"`
}

func main() {
	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize configuration
	var cfg AppConfig

	// Run the service using servicex (logger is automatically created and configured)
	err := servicex.Run(ctx,
		servicex.WithService("greet-service", "0.1.0"),
		servicex.WithConfig(&cfg),
		servicex.WithTracing(true),
		servicex.WithMetrics(true),
		servicex.WithTimeout(30000),
		servicex.WithSlowRequestThreshold(1000),
		servicex.WithShutdownTimeout(15*time.Second),
		servicex.WithRegister(func(app *servicex.App) error {
			// Create Connect handler
			greeterService := &GreeterService{}
			path, handler := greetv1connect.NewGreeterServiceHandler(
				greeterService,
				connect.WithInterceptors(app.Interceptors()...),
			)

			// Register handler
			app.Mux().Handle(path, handler)

			// Use the logger from servicex (automatically configured with service name and context)
			app.Logger().Info("Greet service initialized successfully")
			return nil
		}),
	)
	if err != nil {
		// servicex handles logging internally, but we can still log here if needed
		return
	}
}
