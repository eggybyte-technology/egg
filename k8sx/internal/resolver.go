// Package internal contains Kubernetes service resolver implementation.
package internal

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ServiceResolver resolves Kubernetes services to endpoints.
type ServiceResolver struct {
	client kubernetes.Interface
}

// NewServiceResolver creates a new service resolver.
func NewServiceResolver() (*ServiceResolver, error) {
	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig for local development
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create Kubernetes config: %w", err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	return &ServiceResolver{
		client: client,
	}, nil
}

// ResolveHeadlessService resolves a headless service to individual pod endpoints.
func (r *ServiceResolver) ResolveHeadlessService(ctx context.Context, serviceName string) ([]string, error) {
	name, namespace := parseServiceName(serviceName)

	// Try to get EndpointSlices first (preferred in newer Kubernetes versions)
	endpointSlices, err := r.client.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", name),
	})
	if err == nil && len(endpointSlices.Items) > 0 {
		var endpoints []string
		for _, es := range endpointSlices.Items {
			for _, endpoint := range es.Endpoints {
				if endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready {
					for _, port := range es.Ports {
						if port.Port != nil {
							endpointStr := buildEndpoint(endpoint.Addresses[0], strconv.Itoa(int(*port.Port)))
							endpoints = append(endpoints, endpointStr)
						}
					}
				}
			}
		}
		return endpoints, nil
	}

	// Fallback to Endpoints resource
	endpoints, err := r.client.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints for service %s: %w", serviceName, err)
	}

	var result []string
	for _, subset := range endpoints.Subsets {
		for _, address := range subset.Addresses {
			for _, port := range subset.Ports {
				endpointStr := buildEndpoint(address.IP, strconv.Itoa(int(port.Port)))
				result = append(result, endpointStr)
			}
		}
	}

	return result, nil
}

// ResolveClusterIPService resolves a ClusterIP service to its endpoint.
func (r *ServiceResolver) ResolveClusterIPService(ctx context.Context, serviceName string) ([]string, error) {
	name, namespace := parseServiceName(serviceName)

	// Get the service
	service, err := r.client.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s: %w", serviceName, err)
	}

	if service.Spec.ClusterIP == "" || service.Spec.ClusterIP == "None" {
		return nil, fmt.Errorf("service %s has no ClusterIP", serviceName)
	}

	var endpoints []string
	for _, port := range service.Spec.Ports {
		endpointStr := buildEndpoint(service.Spec.ClusterIP, strconv.Itoa(int(port.Port)))
		endpoints = append(endpoints, endpointStr)
	}

	return endpoints, nil
}

// ResolveService resolves a service based on its type.
func (r *ServiceResolver) ResolveService(ctx context.Context, serviceName string, serviceType string) ([]string, error) {
	switch serviceType {
	case "headless":
		return r.ResolveHeadlessService(ctx, serviceName)
	case "clusterip":
		return r.ResolveClusterIPService(ctx, serviceName)
	default:
		return nil, fmt.Errorf("unknown service type: %s", serviceType)
	}
}

// parseServiceName parses a service name that may include namespace.
// Format: "service" or "service.namespace"
func parseServiceName(serviceName string) (name, namespace string) {
	parts := strings.Split(serviceName, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return serviceName, "default"
}

// buildEndpoint constructs an endpoint string from host and port.
func buildEndpoint(host, port string) string {
	if port == "" {
		return host
	}
	return fmt.Sprintf("%s:%s", host, port)
}
