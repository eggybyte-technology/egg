// Package portproxy provides port proxy management for Docker Compose services.
//
// Overview:
//   - Responsibility: Manage port proxies using socat for Docker Compose services
//   - Key Types: Port proxy manager, port availability checker
//   - Concurrency Model: Sequential proxy management with context support
//   - Error Semantics: Structured errors with port availability information
//   - Performance Notes: Fast port availability checks using net.DialTimeout
//
// Usage:
//
//	manager := NewManager(runner, projectName, networkName)
//	err := manager.CreateProxy(ctx, serviceName, servicePort, localPort)
package portproxy

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"go.eggybyte.com/egg/cli/internal/toolrunner"
)

// Manager provides port proxy management functionality.
type Manager struct {
	runner      *toolrunner.Runner
	projectName string
	networkName string
}

// ProxyInfo represents a port proxy configuration.
type ProxyInfo struct {
	ServiceName string
	ServicePort int
	LocalPort   int
	ProxyName   string
}

// NewManager creates a new port proxy manager.
//
// Parameters:
//   - runner: Tool runner for executing Docker commands
//   - projectName: Project name from configuration
//   - networkName: Docker network name
//
// Returns:
//   - *Manager: Port proxy manager instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewManager(runner *toolrunner.Runner, projectName, networkName string) *Manager {
	return &Manager{
		runner:      runner,
		projectName: projectName,
		networkName: networkName,
	}
}

// CheckPortAvailable checks if a local port is available.
//
// Parameters:
//   - port: Port number to check
//
// Returns:
//   - bool: True if port is available
//   - error: Error if check failed
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Fast timeout-based check (100ms)
func CheckPortAvailable(port int) (bool, error) {
	timeout := 100 * time.Millisecond
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("localhost", strconv.Itoa(port)), timeout)
	if err != nil {
		// Port is likely available if connection failed
		return true, nil
	}
	defer conn.Close()
	return false, nil
}

// FindAvailablePort finds an available port starting from the given port.
//
// Parameters:
//   - startPort: Starting port number to check
//   - maxAttempts: Maximum number of ports to try
//
// Returns:
//   - int: Available port number
//   - error: Error if no available port found
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Linear scan with timeout checks
func FindAvailablePort(startPort, maxAttempts int) (int, error) {
	for i := 0; i < maxAttempts; i++ {
		port := startPort + i
		if port > 65535 {
			return 0, fmt.Errorf("no available port found in range [%d, %d]", startPort, startPort+maxAttempts-1)
		}
		available, err := CheckPortAvailable(port)
		if err != nil {
			continue
		}
		if available {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found in range [%d, %d]", startPort, startPort+maxAttempts-1)
}

// CreateProxy creates a port proxy using socat.
//
// Parameters:
//   - ctx: Context for cancellation
//   - serviceName: Docker Compose service name
//   - servicePort: Port exposed by the service
//   - localPort: Local port to map to (0 to auto-find)
//
// Returns:
//   - *ProxyInfo: Proxy information
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per proxy
//
// Performance:
//   - Docker container creation overhead
func (m *Manager) CreateProxy(ctx context.Context, serviceName string, servicePort int, localPort int) (*ProxyInfo, error) {
	// Determine local port
	finalLocalPort := localPort
	if finalLocalPort == 0 {
		// Try to use the same port as service port first
		available, err := CheckPortAvailable(servicePort)
		if err == nil && available {
			finalLocalPort = servicePort
		} else {
			// Find an available port near the service port
			port, err := FindAvailablePort(servicePort, 100)
			if err != nil {
				return nil, fmt.Errorf("failed to find available port: %w", err)
			}
			finalLocalPort = port
		}
	} else {
		// Check if specified port is available
		available, err := CheckPortAvailable(finalLocalPort)
		if err != nil {
			return nil, fmt.Errorf("failed to check port availability: %w", err)
		}
		if !available {
			// Try to find an alternative port
			port, err := FindAvailablePort(finalLocalPort, 100)
			if err != nil {
				return nil, fmt.Errorf("port %d is not available and no alternative found: %w", finalLocalPort, err)
			}
			finalLocalPort = port
		}
	}

	// Generate proxy container name
	proxyName := fmt.Sprintf("%s-proxy-%s-%d", m.projectName, serviceName, servicePort)

	// Construct Docker service name (with project prefix)
	dockerServiceName := fmt.Sprintf("%s-%s", m.projectName, strings.ReplaceAll(serviceName, "_", "-"))

	// Build socat command
	socatCmd := fmt.Sprintf("tcp-listen:%d,fork,reuseaddr tcp-connect:%s:%d", finalLocalPort, dockerServiceName, servicePort)

	// Run socat container
	args := []string{
		"run", "--rm", "-d",
		"--name", proxyName,
		"-p", fmt.Sprintf("%d:%d", finalLocalPort, finalLocalPort),
		"--network", m.networkName,
		"alpine/socat",
		socatCmd,
	}

	result, err := m.runner.Docker(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to create port proxy: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("docker command failed: %s", result.Stderr)
	}

	return &ProxyInfo{
		ServiceName: serviceName,
		ServicePort: servicePort,
		LocalPort:   finalLocalPort,
		ProxyName:   proxyName,
	}, nil
}

// StopProxy stops a port proxy container.
//
// Parameters:
//   - ctx: Context for cancellation
//   - proxyName: Proxy container name
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded per proxy
//
// Performance:
//   - Docker container stop operation
func (m *Manager) StopProxy(ctx context.Context, proxyName string) error {
	args := []string{"stop", proxyName}
	result, err := m.runner.Docker(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to stop proxy: %w", err)
	}

	if result.ExitCode != 0 {
		// Ignore errors if container doesn't exist
		if !strings.Contains(result.Stderr, "No such container") {
			return fmt.Errorf("docker stop failed: %s", result.Stderr)
		}
	}

	return nil
}

// StopAllProxies stops all port proxy containers for the project.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker container listing and stopping
func (m *Manager) StopAllProxies(ctx context.Context) error {
	// Find all proxy containers
	args := []string{"ps", "--filter", fmt.Sprintf("name=%s-proxy-", m.projectName), "--format", "{{.Names}}"}
	result, err := m.runner.Docker(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to list proxies: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("docker ps failed: %s", result.Stderr)
	}

	// Parse container names
	containerNames := strings.TrimSpace(result.Stdout)
	if containerNames == "" {
		return nil // No proxies running
	}

	names := strings.Split(containerNames, "\n")
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if err := m.StopProxy(ctx, name); err != nil {
			// Continue stopping other proxies even if one fails
			fmt.Printf("Warning: failed to stop proxy %s: %v\n", name, err)
		}
	}

	return nil
}

// ListProxies lists all running port proxy containers for the project.
//
// Parameters:
//   - ctx: Context for cancellation
//
// Returns:
//   - []ProxyInfo: List of proxy information
//   - error: Execution error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Docker container inspection
func (m *Manager) ListProxies(ctx context.Context) ([]ProxyInfo, error) {
	// Find all proxy containers with port information
	args := []string{"ps", "--filter", fmt.Sprintf("name=%s-proxy-", m.projectName), "--format", "{{.Names}}|{{.Ports}}"}
	result, err := m.runner.Docker(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list proxies: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("docker ps failed: %s", result.Stderr)
	}

	// Parse container names and ports
	lines := strings.TrimSpace(result.Stdout)
	if lines == "" {
		return []ProxyInfo{}, nil // No proxies running
	}

	proxyLines := strings.Split(lines, "\n")
	proxies := make([]ProxyInfo, 0, len(proxyLines))

	for _, line := range proxyLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse format: "name|ports"
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		portsStr := strings.TrimSpace(parts[1])

		// Extract service name and port from proxy name
		// Format: project-proxy-service-port
		nameParts := strings.Split(name, "-")
		if len(nameParts) < 3 {
			continue
		}

		// Try to parse port from name (last part)
		portStr := nameParts[len(nameParts)-1]
		servicePort, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		// Reconstruct service name (everything between "proxy" and port)
		serviceName := strings.Join(nameParts[2:len(nameParts)-1], "-")

		// Extract local port from ports string (format: "0.0.0.0:XXXXX->YYYYY/tcp")
		localPort := servicePort // Default to service port
		if portsStr != "" {
			// Parse port mapping: "0.0.0.0:XXXXX->YYYYY/tcp"
			portParts := strings.Split(portsStr, "->")
			if len(portParts) >= 1 {
				hostPortPart := strings.Split(portParts[0], ":")
				if len(hostPortPart) >= 2 {
					hostPortStr := strings.Split(hostPortPart[1], "/")[0]
					if port, err := strconv.Atoi(hostPortStr); err == nil {
						localPort = port
					}
				}
			}
		}

		proxies = append(proxies, ProxyInfo{
			ProxyName:   name,
			ServiceName: serviceName,
			ServicePort: servicePort,
			LocalPort:   localPort,
		})
	}

	return proxies, nil
}
