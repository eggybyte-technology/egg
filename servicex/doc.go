// Package servicex provides a unified microservice initialization framework.
//
// Overview:
//   - Responsibility: Aggregate bootstrap, connectx, configx, obsx into one-call microservice startup
//   - Key Types: Options for configuration, App for service access, Run for startup
//   - Concurrency Model: Safe for concurrent use, supports graceful shutdown
//   - Error Semantics: Initialization errors are wrapped and returned
//   - Performance Notes: Optimized for fast startup with parallel initialization where possible
//
// Usage:
//
//	servicex.Run(ctx, servicex.Options{
//		ServiceName: "my-service",
//		Config:      &AppConfig{},
//		Register: func(app *servicex.App) error {
//			// Register Connect handlers
//			path, handler := greetv1connect.NewGreeterServiceHandler(
//				greeter,
//				connect.WithInterceptors(app.Interceptors()...),
//			)
//			app.Mux().Handle(path, handler)
//			return nil
//		},
//	})
//
// This package replaces the bootstrap module with a more streamlined API that
// automatically handles configuration loading, observability setup, database
// initialization, Connect interceptor configuration, and graceful shutdown.
package servicex
