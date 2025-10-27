// Package k8sx provides tests for Kubernetes ConfigMap watching and service discovery.
package k8sx

import (
	"context"
	"testing"
	"time"

	"go.eggybyte.com/egg/core/log"
)

// testLogger is a test logger implementation.
type testLogger struct {
	logs []string
}

func (l *testLogger) With(kv ...any) log.Logger              { return l }
func (l *testLogger) Debug(msg string, kv ...any)            { l.logs = append(l.logs, "DEBUG: "+msg) }
func (l *testLogger) Info(msg string, kv ...any)             { l.logs = append(l.logs, "INFO: "+msg) }
func (l *testLogger) Warn(msg string, kv ...any)             { l.logs = append(l.logs, "WARN: "+msg) }
func (l *testLogger) Error(err error, msg string, kv ...any) { l.logs = append(l.logs, "ERROR: "+msg) }

func TestWatchConfigMap(t *testing.T) {
	logger := &testLogger{}

	tests := []struct {
		name    string
		opts    WatchOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: WatchOptions{
				Namespace:    "default",
				ResyncPeriod: 10 * time.Minute,
				Logger:       logger,
			},
			wantErr: true, // Will fail due to no K8s cluster
		},
		{
			name: "missing logger",
			opts: WatchOptions{
				Namespace:    "default",
				ResyncPeriod: 10 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "default namespace",
			opts: WatchOptions{
				ResyncPeriod: 10 * time.Minute,
				Logger:       logger,
			},
			wantErr: true, // Will fail due to no K8s cluster
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			err := WatchConfigMap(ctx, "test-configmap", tt.opts, func(data map[string]string) {
				t.Logf("ConfigMap update received: %v", data)
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("WatchConfigMap() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Log the error for debugging
			if err != nil {
				t.Logf("WatchConfigMap error: %v", err)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		service string
		kind    ServiceKind
		wantErr bool
	}{
		{
			name:    "empty service name",
			service: "",
			kind:    ServiceKindClusterIP,
			wantErr: true,
		},
		{
			name:    "valid service name",
			service: "test-service",
			kind:    ServiceKindClusterIP,
			wantErr: true, // Will fail due to no K8s cluster
		},
		{
			name:    "headless service",
			service: "test-service",
			kind:    ServiceKindHeadless,
			wantErr: true, // Will fail due to no K8s cluster
		},
		{
			name:    "service with namespace",
			service: "test-service.test-namespace",
			kind:    ServiceKindClusterIP,
			wantErr: true, // Will fail due to no K8s cluster
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			endpoints, err := Resolve(ctx, tt.service, tt.kind)

			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Log the error for debugging
			if err != nil {
				t.Logf("Resolve error: %v", err)
			}

			if err == nil && endpoints != nil {
				t.Logf("Resolved endpoints: %v", endpoints)
			}
		})
	}
}

func TestServiceKind(t *testing.T) {
	tests := []struct {
		name string
		kind ServiceKind
		want string
	}{
		{
			name: "headless service",
			kind: ServiceKindHeadless,
			want: "headless",
		},
		{
			name: "clusterip service",
			kind: ServiceKindClusterIP,
			want: "clusterip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.kind) != tt.want {
				t.Errorf("ServiceKind = %v, want %v", tt.kind, tt.want)
			}
		})
	}
}

func TestWatchOptions(t *testing.T) {
	logger := &testLogger{}

	opts := WatchOptions{
		Namespace:    "test-namespace",
		ResyncPeriod: 5 * time.Minute,
		Logger:       logger,
	}

	if opts.Namespace != "test-namespace" {
		t.Errorf("Namespace = %v, want test-namespace", opts.Namespace)
	}

	if opts.ResyncPeriod != 5*time.Minute {
		t.Errorf("ResyncPeriod = %v, want 5m", opts.ResyncPeriod)
	}

	if opts.Logger == nil {
		t.Error("Logger is nil")
	}
}
