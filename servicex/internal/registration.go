// Package internal provides registration helpers for simplifying service setup.
//
// This file contains helper functions that simplify common patterns in service
// registration, such as automatic dependency injection resolution and handler registration.

package internal

import (
	"fmt"
	"reflect"

	"go.eggybyte.com/egg/core/log"
	"gorm.io/gorm"
)

// RegisterServices registers common constructors and custom constructors in one call.
//
// This is the recommended way to register services. It combines ProvideCommonConstructors
// and ProvideMany into a single call, reducing boilerplate.
//
// Parameters:
//   - app: Application instance with ProvideTyped, Logger, and DB methods
//   - constructors: Optional map of constructor names to constructor functions (may be nil)
//
// Returns:
//   - error: nil on success; error if registration fails
//
// Usage:
//
//	if err := RegisterServices(app, map[string]any{
//	    "repository": func(db *gorm.DB) repository.UserRepository {
//	        return repository.NewUserRepository(db)
//	    },
//	    "service": func(repo repository.UserRepository, logger log.Logger) service.UserService {
//	        return service.NewUserService(repo, logger)
//	    },
//	}); err != nil {
//	    return err
//	}
func RegisterServices(app interface {
	ProvideTyped(constructor any) error
	Logger() log.Logger
	DB() *gorm.DB
}, constructors map[string]any) error {
	// Register common constructors first (logger, database)
	if err := ProvideCommonConstructors(app); err != nil {
		return fmt.Errorf("failed to register common constructors: %w", err)
	}

	// Register custom constructors if provided
	if len(constructors) > 0 {
		if err := ProvideMany(app, constructors); err != nil {
			return err
		}
	}

	return nil
}

// ProvideCommonConstructors registers common service constructors in the DI container.
//
// This helper registers standard framework dependencies (logger, database) that
// are commonly needed by services. This reduces boilerplate in service registration.
func ProvideCommonConstructors(app interface {
	ProvideTyped(constructor any) error
	Logger() log.Logger
	DB() *gorm.DB
}) error {
	// Register logger (always available)
	if err := app.ProvideTyped(func() log.Logger {
		return app.Logger()
	}); err != nil {
		return fmt.Errorf("failed to register logger constructor: %w", err)
	}

	// Register database (if configured)
	if app.DB() != nil {
		if err := app.ProvideTyped(func() *gorm.DB {
			return app.DB()
		}); err != nil {
			return fmt.Errorf("failed to register database constructor: %w", err)
		}
	}

	return nil
}

// ProvideMany registers multiple constructors, stopping on first error.
//
// This helper simplifies bulk registration of constructors with unified error handling.
// If any constructor fails to register, it returns an error with a descriptive message.
//
// Parameters:
//   - app: Application instance with ProvideTyped method
//   - constructors: Map of constructor names to constructor functions
//
// Returns:
//   - error: nil on success; error with constructor name if registration fails
//
// Usage:
//
//	if err := ProvideMany(app, map[string]any{
//	    "repository": func(db *gorm.DB) repository.UserRepository {
//	        return repository.NewUserRepository(db)
//	    },
//	    "service": func(repo repository.UserRepository, logger log.Logger) service.UserService {
//	        return service.NewUserService(repo, logger, nil)
//	    },
//	}); err != nil {
//	    return err
//	}
func ProvideMany(app interface {
	ProvideTyped(constructor any) error
}, constructors map[string]any) error {
	for name, constructor := range constructors {
		if err := app.ProvideTyped(constructor); err != nil {
			return fmt.Errorf("failed to register %s constructor: %w", name, err)
		}
	}
	return nil
}

// ClientConfig holds configuration for a single client in the registry.
type ClientConfig struct {
	// URLKey is the field name in the config struct to extract URL from
	URLKey string
	// CreateClient is a function that creates the client given URL and internal token
	CreateClient func(url, token string) any
	// ClientName is the name of the client for logging purposes
	ClientName string
}

// ClientRegistryConfig holds configuration for registering multiple optional clients.
//
// This allows batch registration of clients with automatic URL extraction from config.
type ClientRegistryConfig struct {
	// ConfigGetter extracts the config struct from the app
	ConfigGetter func() any
	// Clients is a map of client names to client configurations
	Clients map[string]ClientConfig
}

// RegisterOptionalClients registers multiple optional clients from configuration.
//
// This helper simplifies batch registration of clients by automatically extracting
// URLs from the config struct using reflection. It supports multiple clients and
// provides unified logging.
//
// Parameters:
//   - logger: Logger instance for logging
//   - internalToken: Internal token for service-to-service authentication
//   - config: Client registry configuration
//
// Returns:
//   - map[string]any: Map of client names to created clients (nil for unconfigured clients)
//
// Usage:
//
//	clients := RegisterOptionalClients(logger, app.InternalToken(), ClientRegistryConfig{
//	    ConfigGetter: func() any { return app.Config() },
//	    Clients: map[string]ClientConfig{
//	        "greet": {
//	            URLKey:      "GreetServiceURL",
//	            CreateClient: func(url, token string) any { return client.NewGreetClient(url, token) },
//	            ClientName:   "greet service",
//	        },
//	    },
//	})
func RegisterOptionalClients(logger log.Logger, internalToken string, config ClientRegistryConfig) map[string]any {
	cfg := config.ConfigGetter()
	clients := make(map[string]any)

	for name, clientConfig := range config.Clients {
		// Extract URL from config using reflection
		url := extractURLFromConfig(cfg, clientConfig.URLKey)

		if url == "" {
			logger.Info(fmt.Sprintf("%s client not configured (optional)", clientConfig.ClientName))
			clients[name] = nil
			continue
		}

		client := clientConfig.CreateClient(url, internalToken)
		logger.Info(fmt.Sprintf("%s client initialized", clientConfig.ClientName),
			log.Str("url", url),
			log.Bool("has_token", internalToken != ""))
		clients[name] = client
	}

	return clients
}

// OptionalClientConfig holds configuration for creating an optional client.
type OptionalClientConfig[T any] struct {
	// URLGetter is a function that extracts the URL from the config (legacy, use URLKey instead)
	URLGetter func() string
	// URLKey is the field name in the config struct to extract URL from (preferred, used with RegisterOptionalClients)
	URLKey string
	// CreateClient is a function that creates the client given URL and internal token
	CreateClient func(url, token string) T
	// ClientName is the name of the client for logging purposes (e.g., "greet service")
	ClientName string
}

// extractURLFromConfig extracts a URL field from a config struct using reflection.
func extractURLFromConfig(cfg any, fieldName string) string {
	if cfg == nil {
		return ""
	}

	// Use reflection to get the field value
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return ""
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}

	return field.String()
}

// CreateOptionalClient creates an optional client from configuration and logs the result.
//
// This helper simplifies the common pattern of creating optional service clients
// with consistent logging. If the URL is not configured, it returns nil and logs
// that the client is not configured.
//
// Parameters:
//   - logger: Logger instance for logging
//   - internalToken: Internal token for service-to-service authentication
//   - config: Client configuration
//
// Returns:
//   - T: Created client, or zero value if URL is not configured
//
// Usage:
//
//	greetClient := CreateOptionalClient(logger, app.InternalToken(), OptionalClientConfig[*client.GreetClient]{
//	    URLGetter: func() string {
//	        if cfg, ok := app.Config().(*config.AppConfig); ok && cfg != nil {
//	            return cfg.GreetServiceURL
//	        }
//	        return ""
//	    },
//	    CreateClient: client.NewGreetClient,
//	    ClientName:   "greet service",
//	})
func CreateOptionalClient[T any](logger log.Logger, internalToken string, config OptionalClientConfig[T]) T {
	var zero T
	var url string

	if config.URLGetter != nil {
		url = config.URLGetter()
	}

	if url == "" {
		logger.Info(fmt.Sprintf("%s client not configured (optional)", config.ClientName))
		return zero
	}

	client := config.CreateClient(url, internalToken)
	logger.Info(fmt.Sprintf("%s client initialized", config.ClientName),
		log.Str("url", url),
		log.Bool("has_token", internalToken != ""))

	return client
}
