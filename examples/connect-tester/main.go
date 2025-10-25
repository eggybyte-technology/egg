// Package main provides a Connect service testing tool using egg framework.
//
// This example shows how to:
// - Use logx console format for human-readable output
// - Test Connect RPC services
// - Use clientx for client connections
// - Structure test results
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	greetv1 "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1"
	greetv1connect "github.com/eggybyte-technology/egg/examples/minimal-connect-service/gen/go/greet/v1/greetv1connect"
	"github.com/eggybyte-technology/egg/logx"
)

func main() {
	// Create console logger for human-readable output
	logger := logx.New(
		logx.WithFormat(logx.FormatConsole),
		logx.WithLevel(slog.LevelInfo),
		logx.WithColor(true),
	)

	// Get service URL from command line
	if len(os.Args) < 2 {
		logger.Error(nil, "usage: connect-tester <service-url>")
		os.Exit(1)
	}

	baseURL := os.Args[1]
	logger.Info("connect service tester", "url", baseURL)

	// Run tests
	ctx := context.Background()
	if err := runTests(ctx, logger, baseURL); err != nil {
		logger.Error(err, "tests failed")
		os.Exit(1)
	}

	logger.Info("all tests passed")
}

func runTests(ctx context.Context, logger *logx.Logger, baseURL string) error {
	// Create Connect client
	client := greetv1connect.NewGreeterServiceClient(
		http.DefaultClient,
		baseURL,
	)

	logger.Info("testing SayHello endpoint")
	if err := testSayHello(ctx, logger, client); err != nil {
		return err
	}

	logger.Info("testing SayHelloStream endpoint")
	if err := testSayHelloStream(ctx, logger, client); err != nil {
		return err
	}

	return nil
}

func testSayHello(ctx context.Context, logger *logx.Logger, client greetv1connect.GreeterServiceClient) error {
	start := time.Now()

	req := connect.NewRequest(&greetv1.SayHelloRequest{
		Name:     "Tester",
		Language: "en",
	})

	resp, err := client.SayHello(ctx, req)
	if err != nil {
		logger.Error(err, "SayHello failed")
		return err
	}

	duration := time.Since(start)
	logger.Info("SayHello success",
		"message", resp.Msg.Message,
		"duration", fmt.Sprintf("%dms", duration.Milliseconds()),
	)

	return nil
}

func testSayHelloStream(ctx context.Context, logger *logx.Logger, client greetv1connect.GreeterServiceClient) error {
	start := time.Now()

	req := connect.NewRequest(&greetv1.SayHelloStreamRequest{
		Name:  "Tester",
		Count: 3,
	})

	stream, err := client.SayHelloStream(ctx, req)
	if err != nil {
		logger.Error(err, "SayHelloStream failed")
		return err
	}

	var messages []string
	for stream.Receive() {
		messages = append(messages, stream.Msg().Message)
	}

	if err := stream.Err(); err != nil {
		logger.Error(err, "stream receive failed")
		return err
	}

	duration := time.Since(start)
	logger.Info("SayHelloStream success",
		"messages", len(messages),
		"duration", fmt.Sprintf("%dms", duration.Milliseconds()),
	)

	return nil
}
