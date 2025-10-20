# K8sX Module

<div align="center">

**Kubernetes integration for Egg services**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue.svg)](https://golang.org/dl/)
[![License](https://img.shields.io/badge/License-Apache%202.0-green.svg)](LICENSE)

</div>

## 📦 Overview

The `k8sx` module provides Kubernetes integration for Egg services. It offers ConfigMap name-based watching, service discovery, and Secret contracts for seamless Kubernetes-native development.

## ✨ Features

- ☸️ **ConfigMap Watching** - Name-based ConfigMap watching with hot reload
- 🔍 **Service Discovery** - Headless and ClusterIP service discovery
- 🔐 **Secret Contracts** - Secret injection via env + secretKeyRef
- 🔄 **Hot Reload** - Configuration changes without service restart
- 📝 **Structured Logging** - Context-aware logging
- 🛡️ **Error Handling** - Robust error handling and recovery
- 🎯 **Kubernetes Native** - Designed for Kubernetes environments
- 🔧 **Easy Configuration** - Simple setup and configuration

## 🏗️ Architecture

```
k8sx/
├── k8sx.go           # Main Kubernetes integration
├── internal/
│   ├── watcher.go    # ConfigMap watcher
│   └── resolver.go   # Service resolver
└── k8sx_test.go      # Tests
```

## 🚀 Quick Start

### Installation

```bash
go get github.com/eggybyte-technology/egg/k8sx@latest
```

### Basic Usage

```go
package main

import (
    "context"
    "time"

    "github.com/eggybyte-technology/egg/k8sx"
    "github.com/eggybyte-technology/egg/core/log"
)

func main() {
    // Create logger
    logger := &YourLogger{} // Implement log.Logger interface

    // Create Kubernetes client
    ctx := context.Background()
    client, err := k8sx.NewClient(ctx, k8sx.Options{
        Namespace: "default",
        Logger:    logger,
    })
    if err != nil {
        log.Fatal("Failed to create Kubernetes client:", err)
    }
    defer client.Close()

    // Watch ConfigMap
    configMapName := "my-service-config"
    if err := client.WatchConfigMap(ctx, configMapName, func(data map[string]string) {
        log.Info("ConfigMap updated", "name", configMapName, "data", data)
        // Handle configuration changes
        handleConfigChange(data)
    }); err != nil {
        log.Fatal("Failed to watch ConfigMap:", err)
    }

    // Resolve service
    serviceName := "my-service"
    endpoints, err := client.ResolveService(ctx, serviceName, k8sx.ServiceTypeClusterIP)
    if err != nil {
        log.Fatal("Failed to resolve service:", err)
    }

    log.Info("Service resolved", "service", serviceName, "endpoints", endpoints)
}
```

## 📖 API Reference

### Client Options

```go
type Options struct {
    Namespace string
    Logger    log.Logger
    KubeConfig string
    InCluster  bool
}

type Client interface {
    WatchConfigMap(ctx context.Context, name string, callback func(map[string]string)) error
    ResolveService(ctx context.Context, name string, serviceType ServiceType) ([]string, error)
    GetSecret(ctx context.Context, name string) (map[string][]byte, error)
    Close() error
}
```

### Service Types

```go
type ServiceType int

const (
    ServiceTypeClusterIP ServiceType = iota
    ServiceTypeHeadless
)
```

### Main Functions

```go
// NewClient creates a new Kubernetes client
func NewClient(ctx context.Context, opts Options) (Client, error)

// DefaultClient creates a client with default options
func DefaultClient(ctx context.Context, namespace string) (Client, error)
```

## 🔧 Configuration

### Environment Variables

```bash
# Kubernetes configuration
export NAMESPACE="default"
export KUBECONFIG="/path/to/kubeconfig"

# Service discovery
export SERVICE_NAME="my-service"
export SERVICE_TYPE="clusterip"

# ConfigMap watching
export CONFIGMAP_NAME="my-service-config"
```

### Kubernetes RBAC

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: my-service
  namespace: default
rules:
- apiGroups: [""]
  resources: ["configmaps", "secrets", "services", "endpoints"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: my-service
  namespace: default
subjects:
- kind: ServiceAccount
  name: my-service
  namespace: default
roleRef:
  kind: Role
  name: my-service
  apiGroup: rbac.authorization.k8s.io
```

## 🛠️ Advanced Usage

### ConfigMap Watching

```go
func main() {
    // Create client
    client, err := k8sx.NewClient(ctx, k8sx.Options{
        Namespace: "default",
        Logger:    logger,
    })
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // Watch multiple ConfigMaps
    configMaps := []string{
        "my-service-config",
        "shared-config",
        "feature-flags",
    }

    for _, name := range configMaps {
        go func(configMapName string) {
            if err := client.WatchConfigMap(ctx, configMapName, func(data map[string]string) {
                log.Info("ConfigMap updated", "name", configMapName)
                // Handle specific ConfigMap changes
                handleConfigMapChange(configMapName, data)
            }); err != nil {
                log.Error("Failed to watch ConfigMap", "name", configMapName, "error", err)
            }
        }(name)
    }

    // Keep running
    select {}
}
```

### Service Discovery

```go
func main() {
    // Create client
    client, err := k8sx.NewClient(ctx, k8sx.Options{
        Namespace: "default",
        Logger:    logger,
    })
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // Resolve ClusterIP service
    endpoints, err := client.ResolveService(ctx, "my-service", k8sx.ServiceTypeClusterIP)
    if err != nil {
        log.Fatal("Failed to resolve service:", err)
    }

    log.Info("ClusterIP endpoints", "endpoints", endpoints)

    // Resolve Headless service
    headlessEndpoints, err := client.ResolveService(ctx, "my-service", k8sx.ServiceTypeHeadless)
    if err != nil {
        log.Fatal("Failed to resolve headless service:", err)
    }

    log.Info("Headless endpoints", "endpoints", headlessEndpoints)
}
```

### Secret Management

```go
func main() {
    // Create client
    client, err := k8sx.NewClient(ctx, k8sx.Options{
        Namespace: "default",
        Logger:    logger,
    })
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // Get secret
    secretName := "my-service-secrets"
    secrets, err := client.GetSecret(ctx, secretName)
    if err != nil {
        log.Fatal("Failed to get secret:", err)
    }

    // Use secrets
    for key, value := range secrets {
        log.Info("Secret loaded", "key", key, "length", len(value))
        // Use secret value (be careful not to log sensitive data)
    }
}
```

## 🔧 Integration with Other Modules

### ConfigX Integration

```go
func main() {
    // Create Kubernetes client
    client, err := k8sx.NewClient(ctx, k8sx.Options{
        Namespace: "default",
        Logger:    logger,
    })
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // Create configuration manager with Kubernetes source
    mgr, err := configx.NewManager(ctx, logger,
        configx.WithSources(
            configx.EnvironmentSource(),
            configx.FileSource("config.yaml"),
            configx.ConfigMapSource("my-service-config", "default"),
        ),
    )
    if err != nil {
        log.Fatal("Failed to create config manager:", err)
    }

    // Bind configuration
    var cfg AppConfig
    if err := mgr.Bind(&cfg); err != nil {
        log.Fatal("Failed to bind configuration:", err)
    }

    // Watch for configuration changes
    if err := mgr.Watch(ctx, func() {
        log.Info("Configuration reloaded from Kubernetes")
    }); err != nil {
        log.Fatal("Failed to watch configuration:", err)
    }
}
```

### Service Discovery Integration

```go
func main() {
    // Create client
    client, err := k8sx.NewClient(ctx, k8sx.Options{
        Namespace: "default",
        Logger:    logger,
    })
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }
    defer client.Close()

    // Resolve dependencies
    dependencies := []string{
        "user-service",
        "order-service",
        "payment-service",
    }

    for _, serviceName := range dependencies {
        endpoints, err := client.ResolveService(ctx, serviceName, k8sx.ServiceTypeClusterIP)
        if err != nil {
            log.Error("Failed to resolve service", "service", serviceName, "error", err)
            continue
        }

        log.Info("Service resolved", "service", serviceName, "endpoints", endpoints)
        // Use endpoints for service communication
    }
}
```

## 🧪 Testing

Run tests:

```bash
go test ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

## 📈 Test Coverage

| Component | Coverage |
|-----------|----------|
| K8sX | Good |

## 🔍 Troubleshooting

### Common Issues

1. **Kubernetes Client Not Connected**
   ```bash
   # Check if kubeconfig is valid
   kubectl cluster-info
   
   # Check if service account has permissions
   kubectl auth can-i get configmaps --as=system:serviceaccount:default:my-service
   ```

2. **ConfigMap Not Found**
   ```bash
   # Check if ConfigMap exists
   kubectl get configmap my-service-config -n default
   
   # Check if service account can access ConfigMap
   kubectl auth can-i get configmap my-service-config --as=system:serviceaccount:default:my-service
   ```

3. **Service Resolution Failed**
   ```bash
   # Check if service exists
   kubectl get service my-service -n default
   
   # Check service endpoints
   kubectl get endpoints my-service -n default
   ```

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
