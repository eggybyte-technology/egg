// Package internal contains the runtime implementation.
package internal

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/eggybyte-technology/egg/core/log"
)

// Runtime manages the lifecycle of services and servers.
type Runtime struct {
	logger          log.Logger
	httpServer      *http.Server
	rpcServer       *http.Server
	healthServer    *http.Server
	metricsServer   *http.Server
	services        []Service
	shutdownTimeout time.Duration
}

// Service is the interface for services that can be started and stopped.
type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// NewRuntime creates a new runtime instance.
func NewRuntime(logger log.Logger, services []Service, shutdownTimeout time.Duration) *Runtime {
	return &Runtime{
		logger:          logger,
		services:        services,
		shutdownTimeout: shutdownTimeout,
	}
}

// Start starts all services and servers.
func (r *Runtime) Start(ctx context.Context) error {
	r.logger.Info("starting runtime")

	// Start services concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, len(r.services))

	for i, service := range r.services {
		wg.Add(1)
		go func(idx int, svc Service) {
			defer wg.Done()
			r.logger.Info("starting service", log.Int("index", idx))
			if err := svc.Start(ctx); err != nil {
				r.logger.Error(err, "service start failed", log.Int("index", idx))
				errChan <- fmt.Errorf("service %d start failed: %w", idx, err)
			} else {
				r.logger.Info("service started", log.Int("index", idx))
			}
		}(i, service)
	}

	// Wait for all services to start or fail
	wg.Wait()
	close(errChan)

	// Check for startup errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	// Start HTTP server if configured
	if r.httpServer != nil {
		go func() {
			r.logger.Info("starting HTTP server", log.Str("addr", r.httpServer.Addr))
			if err := r.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				r.logger.Error(err, "HTTP server failed")
			}
		}()
	}

	// Start RPC server if configured
	if r.rpcServer != nil {
		go func() {
			r.logger.Info("starting RPC server", log.Str("addr", r.rpcServer.Addr))
			if err := r.rpcServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				r.logger.Error(err, "RPC server failed")
			}
		}()
	}

	// Start health server if configured
	if r.healthServer != nil {
		go func() {
			r.logger.Info("starting health server", log.Str("addr", r.healthServer.Addr))
			if err := r.healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				r.logger.Error(err, "health server failed")
			}
		}()
	}

	// Start metrics server if configured
	if r.metricsServer != nil {
		go func() {
			r.logger.Info("starting metrics server", log.Str("addr", r.metricsServer.Addr))
			if err := r.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				r.logger.Error(err, "metrics server failed")
			}
		}()
	}

	r.logger.Info("runtime started successfully")
	return nil
}

// Stop gracefully shuts down all services and servers.
func (r *Runtime) Stop(ctx context.Context) error {
	r.logger.Info("stopping runtime")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, r.shutdownTimeout)
	defer cancel()

	// Stop services concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, len(r.services))

	for i, service := range r.services {
		wg.Add(1)
		go func(idx int, svc Service) {
			defer wg.Done()
			r.logger.Info("stopping service", log.Int("index", idx))
			if err := svc.Stop(shutdownCtx); err != nil {
				r.logger.Error(err, "service stop failed", log.Int("index", idx))
				errChan <- fmt.Errorf("service %d stop failed: %w", idx, err)
			} else {
				r.logger.Info("service stopped", log.Int("index", idx))
			}
		}(i, service)
	}

	// Wait for all services to stop
	wg.Wait()
	close(errChan)

	// Check for shutdown errors
	for err := range errChan {
		if err != nil {
			r.logger.Error(err, "service shutdown error")
		}
	}

	// Stop servers
	if r.httpServer != nil {
		r.logger.Info("stopping HTTP server")
		if err := r.httpServer.Shutdown(shutdownCtx); err != nil {
			r.logger.Error(err, "HTTP server shutdown failed")
		}
	}

	if r.rpcServer != nil {
		r.logger.Info("stopping RPC server")
		if err := r.rpcServer.Shutdown(shutdownCtx); err != nil {
			r.logger.Error(err, "RPC server shutdown failed")
		}
	}

	if r.healthServer != nil {
		r.logger.Info("stopping health server")
		if err := r.healthServer.Shutdown(shutdownCtx); err != nil {
			r.logger.Error(err, "health server shutdown failed")
		}
	}

	if r.metricsServer != nil {
		r.logger.Info("stopping metrics server")
		if err := r.metricsServer.Shutdown(shutdownCtx); err != nil {
			r.logger.Error(err, "metrics server shutdown failed")
		}
	}

	r.logger.Info("runtime stopped")
	return nil
}

// SetHTTPServer sets the HTTP server.
func (r *Runtime) SetHTTPServer(server *http.Server) {
	r.httpServer = server
}

// SetRPCServer sets the RPC server.
func (r *Runtime) SetRPCServer(server *http.Server) {
	r.rpcServer = server
}

// SetHealthServer sets the health server.
func (r *Runtime) SetHealthServer(server *http.Server) {
	r.healthServer = server
}

// SetMetricsServer sets the metrics server.
func (r *Runtime) SetMetricsServer(server *http.Server) {
	r.metricsServer = server
}
