# Core Module

<div align="center">

**Zero-dependency core interfaces and utilities for Egg framework**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

</div>

## 📦 Overview

The `core` module provides zero-dependency interfaces and utilities that form the foundation of the Egg framework. These modules have no external dependencies and can be used independently.

## 🏗️ Architecture

```
core/
├── log/        # Structured logging interface
├── errors/     # Error handling with codes and wrapping
├── identity/   # User identity and request metadata
└── utils/      # Common utilities (retry, time, slices)
```

## 📚 Modules

### `log` - Logging Interface

Provides a structured logging interface compatible with Go's `slog` philosophy.

**Key Features:**
- Structured logging with key-value pairs
- Compatible with `log/slog` interface
- Zero dependencies
- Context-aware logging

**Example Usage:**

```go
import "go.eggybyte.com/egg/core/log"

type Logger struct {
    // Your implementation
}

func (l *Logger) Log(ctx context.Context, level log.Level, msg string, attrs ...log.Attr) {
    // Implementation
}

// Use with slog
logger := &Logger{}
slog.SetDefault(slog.New(log.NewHandler(logger)))
```

### `errors` - Error Handling

Layered error handling with error codes and wrapping support.

**Key Features:**
- Error codes for categorization
- Error wrapping with context
- Structured error information
- Zero dependencies

**Example Usage:**

```go
import "go.eggybyte.com/egg/core/errors"

// Define error codes
const (
    ErrCodeNotFound = "NOT_FOUND"
    ErrCodeInvalidInput = "INVALID_INPUT"
)

// Create errors
err := errors.New(ErrCodeNotFound, "user not found")
wrappedErr := errors.Wrap(err, "failed to get user")

// Check error codes
if errors.IsCode(err, ErrCodeNotFound) {
    // Handle not found
}
```

### `identity` - User Identity

User identity and request metadata container for authentication and authorization.

**Key Features:**
- User identity extraction
- Request metadata storage
- Context propagation
- Zero dependencies

**Example Usage:**

```go
import "go.eggybyte.com/egg/core/identity"

// Extract identity from context
userID, ok := identity.UserIDFromContext(ctx)
if !ok {
    return errors.New("UNAUTHORIZED", "user not authenticated")
}

// Get request metadata
reqID := identity.RequestIDFromContext(ctx)
traceID := identity.TraceIDFromContext(ctx)
```

### `utils` - Common Utilities

Common utilities for retry logic, time operations, and slice manipulations.

**Key Features:**
- Retry logic with exponential backoff
- Time utilities
- Slice operations
- Zero dependencies

**Example Usage:**

```go
import "go.eggybyte.com/egg/core/utils"

// Retry with exponential backoff
err := utils.Retry(ctx, 3, time.Second, func() error {
    return someOperation()
})

// Slice utilities
filtered := utils.Filter(slice, func(item string) bool {
    return len(item) > 0
})
```

## 🚀 Quick Start

### Installation

```bash
go get go.eggybyte.com/egg/core@latest
```

### Basic Usage

```go
package main

import (
    "context"
    "log/slog"
    
    "go.eggybyte.com/egg/core/log"
    "go.eggybyte.com/egg/core/errors"
    "go.eggybyte.com/egg/core/identity"
    "go.eggybyte.com/egg/core/utils"
)

func main() {
    // Set up logging
    logger := slog.Default()
    
    // Create context with identity
    ctx := context.Background()
    ctx = identity.WithUserID(ctx, "user123")
    ctx = identity.WithRequestID(ctx, "req456")
    
    // Log with context
    logger.InfoContext(ctx, "processing request")
    
    // Handle errors
    if err := processRequest(ctx); err != nil {
        logger.ErrorContext(ctx, "request failed", "error", err)
    }
}

func processRequest(ctx context.Context) error {
    // Retry logic
    return utils.Retry(ctx, 3, time.Second, func() error {
        // Your operation
        return nil
    })
}
```

## 📖 API Reference

### Logging Interface

```go
type Logger interface {
    Log(ctx context.Context, level Level, msg string, attrs ...Attr)
}

type Level int

const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
)

type Attr struct {
    Key   string
    Value interface{}
}
```

### Error Handling

```go
func New(code, message string) error
func Wrap(err error, message string) error
func IsCode(err error, code string) bool
func Code(err error) string
```

### Identity Context

```go
func WithUserID(ctx context.Context, userID string) context.Context
func UserIDFromContext(ctx context.Context) (string, bool)
func WithRequestID(ctx context.Context, reqID string) context.Context
func RequestIDFromContext(ctx context.Context) (string, bool)
func WithTraceID(ctx context.Context, traceID string) context.Context
func TraceIDFromContext(ctx context.Context) (string, bool)
```

### Utilities

```go
func Retry(ctx context.Context, maxAttempts int, baseDelay time.Duration, fn func() error) error
func Filter[T any](slice []T, predicate func(T) bool) []T
func Map[T, U any](slice []T, mapper func(T) U) []U
func Contains[T comparable](slice []T, item T) bool
```

## 🧪 Testing

Run tests for all core modules:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

## 📈 Test Coverage

| Module | Coverage |
|--------|----------|
| log | 100% |
| errors | 91.7% |
| identity | 100% |
| utils | 94.3% |

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](../../LICENSE) file for details.

---

<div align="center">

**Built with ❤️ by EggyByte Technology**

</div>
