// Demo script to show egg log format with colors
package main

import (
	"log/slog"
	"os"

	"github.com/eggybyte-technology/egg/logx"
)

func main() {
	// Create logger with colors enabled
	logger := logx.New(
		logx.WithFormat(logx.FormatLogfmt),
		logx.WithLevel(slog.LevelDebug),
		logx.WithColor(true),
		logx.WithWriter(os.Stdout),
	)

	// Add service context
	logger = logger.With(
		"service", "demo-service",
		"version", "1.0.0",
	)

	println("\n=== Egg Log Format Demo (With Colors) ===\n")

	// Demonstrate different log levels with proper coloring
	logger.Debug("debug message", "op", "initialization", "status", "starting")

	logger.Info("starting service")

	logger.Info("configuration loaded", "keys", 45, "env", "production")

	logger.Info("request started",
		"procedure", "/user.v1.UserService/CreateUser",
		"request_id", "req-abc123",
	)

	logger.Info("user created successfully",
		"user_id", "u-12345",
		"email", "test@example.com",
		"duration_ms", 1.5,
	)

	logger.Warn("slow request detected",
		"procedure", "/user.v1.UserService/ListUsers",
		"duration_ms", 1523,
		"threshold_ms", 1000,
	)

	logger.Error(nil, "database query failed",
		"op", "repository.GetUser",
		"code", "NOT_FOUND",
		"error", "record not found",
	)

	logger.Info("request completed",
		"procedure", "/user.v1.UserService/CreateUser",
		"duration_ms", 2.3,
		"status", "OK",
	)

	println("\n=== Egg Log Format Demo (Without Colors) ===\n")

	// Create logger without colors
	loggerNoColor := logx.New(
		logx.WithFormat(logx.FormatLogfmt),
		logx.WithLevel(slog.LevelInfo),
		logx.WithColor(false),
		logx.WithWriter(os.Stdout),
	)

	loggerNoColor = loggerNoColor.With(
		"service", "demo-service",
		"version", "1.0.0",
	)

	loggerNoColor.Info("starting service")
	loggerNoColor.Info("configuration loaded", "keys", 45)
	loggerNoColor.Warn("warning message", "threshold", 1000)
	loggerNoColor.Error(nil, "error message", "code", "INTERNAL", "op", "test")

	println("\n=== Key Features ===")
	println("✓ Single-line logfmt format")
	println("✓ Only level field is colored")
	println("✓ Fields sorted alphabetically (after level and msg)")
	println("✓ No timestamps (containers add them)")
	println("✓ Service and version automatically injected")
	println("✓ Loki/Promtail compatible")
	println("✓ All strings properly quoted")
	println()
}
