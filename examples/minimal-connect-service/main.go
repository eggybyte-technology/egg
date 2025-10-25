// Package main demonstrates a minimal Connect service using the servicex library.
//
// This example shows how to:
// - Set up a Connect service with minimal code using servicex
// - Use servicex for unified service initialization with console logging
// - Register a simple Connect RPC handler
// - Handle graceful shutdown automatically
// - Configure health and metrics endpoints
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

// GreeterService implements the greeting service
type GreeterService struct {
	logger log.Logger
}

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

// registerServices registers all service handlers
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
