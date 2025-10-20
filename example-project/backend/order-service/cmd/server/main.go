package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/eggybyte-technology/egg/runtimex"
	"github.com/eggybyte-technology/egg/connectx"
	"github.com/eggybyte-technology/egg/configx"
	"github.com/eggybyte-technology/example-platform/backend/order-service/internal/config"
)

func main() {
	ctx := context.Background()
	var cfg config.AppConfig
	
	// Initialize configuration manager
	_, err := configx.QuickBind(ctx, nil, &cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to bind config: %v", err))
	}

	mux := http.NewServeMux()
	_ = connectx.DefaultInterceptors(connectx.Options{SlowRequestMillis: 1000})
	
	// TODO: Add your Connect handlers here
	// path, h := yourapi.NewServiceHandler(..., connect.WithInterceptors(ints...))
	// mux.Handle(path, h)

	_ = runtimex.Run(ctx, nil, runtimex.Options{
		HTTP:    &runtimex.HTTPOptions{Addr: ":" + cfg.HTTPPort, H2C: true, Mux: mux},
		Health:  &runtimex.Endpoint{Addr: ":" + cfg.HealthPort},
		Metrics: &runtimex.Endpoint{Addr: ":" + cfg.MetricsPort},
		ShutdownTimeout: 15 * time.Second,
	})
}
