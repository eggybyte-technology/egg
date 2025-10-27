// Package helm provides Helm chart rendering for egg projects.
//
// Overview:
//   - Responsibility: Render Helm charts from egg configuration
//   - Key Types: Helm renderer, chart templates, Kubernetes manifests
//   - Concurrency Model: Immutable rendering with atomic file writes
//   - Error Semantics: Rendering errors with configuration validation
//   - Performance Notes: Template-based rendering, minimal I/O operations
//
// Usage:
//
//	renderer := NewRenderer(fs, refParser)
//	err := renderer.Render(config)
package helm

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"go.eggybyte.com/egg/cli/internal/configschema"
	"go.eggybyte.com/egg/cli/internal/projectfs"
	"go.eggybyte.com/egg/cli/internal/ref"
	"go.eggybyte.com/egg/cli/internal/ui"
)

// Renderer provides Helm chart rendering functionality.
//
// Parameters:
//   - fs: Project file system
//   - refParser: Reference expression parser
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Template-based rendering
type Renderer struct {
	fs        *projectfs.ProjectFS
	refParser *ref.Parser
}

// NewRenderer creates a new Helm renderer.
//
// Parameters:
//   - fs: Project file system
//   - refParser: Reference expression parser
//
// Returns:
//   - *Renderer: Helm renderer instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewRenderer(fs *projectfs.ProjectFS, refParser *ref.Parser) *Renderer {
	return &Renderer{
		fs:        fs,
		refParser: refParser,
	}
}

// Render renders Helm charts for all services.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - error: Rendering error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering and file I/O
func (r *Renderer) Render(config *configschema.Config) error {
	ui.Info("Rendering Helm charts...")

	// Create helm directory structure
	if err := r.fs.CreateDirectory("deploy/helm"); err != nil {
		return fmt.Errorf("failed to create helm directory: %w", err)
	}

	// Render backend services
	for name, service := range config.Backend {
		if err := r.renderBackendService(name, service, config); err != nil {
			return fmt.Errorf("failed to render backend service %s: %w", name, err)
		}
	}

	// Render frontend services
	for name, service := range config.Frontend {
		if err := r.renderFrontendService(name, service, config); err != nil {
			return fmt.Errorf("failed to render frontend service %s: %w", name, err)
		}
	}

	// Render database service if enabled
	if config.Database.Enabled {
		if err := r.renderDatabaseService(config.Database, config); err != nil {
			return fmt.Errorf("failed to render database service: %w", err)
		}
	}

	// Render ConfigMaps and Secrets
	if err := r.renderKubernetesResources(config); err != nil {
		return fmt.Errorf("failed to render Kubernetes resources: %w", err)
	}

	ui.Success("Helm charts rendered")
	return nil
}

// renderBackendService renders a backend service Helm chart.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - error: Rendering error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering and file I/O
func (r *Renderer) renderBackendService(name string, service configschema.BackendService, config *configschema.Config) error {
	// Create service directory
	serviceDir := filepath.Join("deploy/helm", name)
	if err := r.fs.CreateDirectory(serviceDir); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Create templates directory
	templatesDir := filepath.Join(serviceDir, "templates")
	if err := r.fs.CreateDirectory(templatesDir); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Generate Chart.yaml
	chartYAML := r.generateChartYAML(name, config)
	if err := r.fs.WriteFile(filepath.Join(serviceDir, "Chart.yaml"), chartYAML, 0644); err != nil {
		return fmt.Errorf("failed to write Chart.yaml: %w", err)
	}

	// Generate values.yaml
	valuesYAML, err := r.generateBackendValues(name, service, config)
	if err != nil {
		return fmt.Errorf("failed to generate values.yaml: %w", err)
	}
	if err := r.fs.WriteFile(filepath.Join(serviceDir, "values.yaml"), valuesYAML, 0644); err != nil {
		return fmt.Errorf("failed to write values.yaml: %w", err)
	}

	// Generate Deployment template
	deploymentYAML, err := r.generateBackendDeployment(name, service, config)
	if err != nil {
		return fmt.Errorf("failed to generate deployment template: %w", err)
	}
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "deployment.yaml"), deploymentYAML, 0644); err != nil {
		return fmt.Errorf("failed to write deployment.yaml: %w", err)
	}

	// Generate Service templates
	if err := r.generateBackendServices(name, service, config, templatesDir); err != nil {
		return fmt.Errorf("failed to generate service templates: %w", err)
	}

	// Generate ConfigMap template
	configMapYAML, err := r.generateBackendConfigMap(name, service, config)
	if err != nil {
		return fmt.Errorf("failed to generate configmap template: %w", err)
	}
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "configmap.yaml"), configMapYAML, 0644); err != nil {
		return fmt.Errorf("failed to write configmap.yaml: %w", err)
	}

	return nil
}

// renderFrontendService renders a frontend service Helm chart.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - error: Rendering error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering and file I/O
func (r *Renderer) renderFrontendService(name string, service configschema.FrontendService, config *configschema.Config) error {
	// Create service directory
	serviceDir := filepath.Join("deploy/helm", name)
	if err := r.fs.CreateDirectory(serviceDir); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Create templates directory
	templatesDir := filepath.Join(serviceDir, "templates")
	if err := r.fs.CreateDirectory(templatesDir); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Generate Chart.yaml
	chartYAML := r.generateChartYAML(name, config)
	if err := r.fs.WriteFile(filepath.Join(serviceDir, "Chart.yaml"), chartYAML, 0644); err != nil {
		return fmt.Errorf("failed to write Chart.yaml: %w", err)
	}

	// Generate values.yaml
	valuesYAML := r.generateFrontendValues(name, service, config)
	if err := r.fs.WriteFile(filepath.Join(serviceDir, "values.yaml"), valuesYAML, 0644); err != nil {
		return fmt.Errorf("failed to write values.yaml: %w", err)
	}

	// Generate Deployment template
	deploymentYAML := r.generateFrontendDeployment(name, service, config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "deployment.yaml"), deploymentYAML, 0644); err != nil {
		return fmt.Errorf("failed to write deployment.yaml: %w", err)
	}

	// Generate Service template
	serviceYAML := r.generateFrontendService(name, service, config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "service.yaml"), serviceYAML, 0644); err != nil {
		return fmt.Errorf("failed to write service.yaml: %w", err)
	}

	return nil
}

// renderDatabaseService renders the database service Helm chart.
//
// Parameters:
//   - db: Database configuration
//   - config: Project configuration
//
// Returns:
//   - error: Rendering error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering and file I/O
func (r *Renderer) renderDatabaseService(db configschema.DatabaseConfig, config *configschema.Config) error {
	// Create service directory
	serviceDir := filepath.Join("deploy/helm", "mysql")
	if err := r.fs.CreateDirectory(serviceDir); err != nil {
		return fmt.Errorf("failed to create service directory: %w", err)
	}

	// Create templates directory
	templatesDir := filepath.Join(serviceDir, "templates")
	if err := r.fs.CreateDirectory(templatesDir); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Generate Chart.yaml
	chartYAML := r.generateChartYAML("mysql", config)
	if err := r.fs.WriteFile(filepath.Join(serviceDir, "Chart.yaml"), chartYAML, 0644); err != nil {
		return fmt.Errorf("failed to write Chart.yaml: %w", err)
	}

	// Generate values.yaml
	valuesYAML := r.generateDatabaseValues(db, config)
	if err := r.fs.WriteFile(filepath.Join(serviceDir, "values.yaml"), valuesYAML, 0644); err != nil {
		return fmt.Errorf("failed to write values.yaml: %w", err)
	}

	// Generate Deployment template
	deploymentYAML := r.generateDatabaseDeployment(db, config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "deployment.yaml"), deploymentYAML, 0644); err != nil {
		return fmt.Errorf("failed to write deployment.yaml: %w", err)
	}

	// Generate Service template
	serviceYAML := r.generateDatabaseService(db, config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "service.yaml"), serviceYAML, 0644); err != nil {
		return fmt.Errorf("failed to write service.yaml: %w", err)
	}

	// Generate Secret template
	secretYAML := r.generateDatabaseSecret(db, config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "secret.yaml"), secretYAML, 0644); err != nil {
		return fmt.Errorf("failed to write secret.yaml: %w", err)
	}

	return nil
}

// renderKubernetesResources renders ConfigMaps and Secrets.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - error: Rendering error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering and file I/O
func (r *Renderer) renderKubernetesResources(config *configschema.Config) error {
	// Create resources directory
	resourcesDir := filepath.Join("deploy/helm", "resources")
	if err := r.fs.CreateDirectory(resourcesDir); err != nil {
		return fmt.Errorf("failed to create resources directory: %w", err)
	}

	// Create templates directory
	templatesDir := filepath.Join(resourcesDir, "templates")
	if err := r.fs.CreateDirectory(templatesDir); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Generate Chart.yaml
	chartYAML := r.generateChartYAML("resources", config)
	if err := r.fs.WriteFile(filepath.Join(resourcesDir, "Chart.yaml"), chartYAML, 0644); err != nil {
		return fmt.Errorf("failed to write Chart.yaml: %w", err)
	}

	// Generate ConfigMap templates
	for name, data := range config.Kubernetes.Resources.ConfigMaps {
		configMapYAML := r.generateConfigMap(name, data, config)
		filename := fmt.Sprintf("configmap-%s.yaml", name)
		if err := r.fs.WriteFile(filepath.Join(templatesDir, filename), configMapYAML, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	// Generate Secret templates
	for name, data := range config.Kubernetes.Resources.Secrets {
		secretYAML := r.generateSecret(name, data, config)
		filename := fmt.Sprintf("secret-%s.yaml", name)
		if err := r.fs.WriteFile(filepath.Join(templatesDir, filename), secretYAML, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}

// generateChartYAML generates Chart.yaml content.
//
// Parameters:
//   - name: Chart name
//   - config: Project configuration
//
// Returns:
//   - string: Chart.yaml content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) generateChartYAML(name string, config *configschema.Config) string {
	return fmt.Sprintf(`apiVersion: v2
name: %s
description: A Helm chart for %s
type: application
version: 0.1.0
appVersion: "%s"
`, name, name, config.Version)
}

// generateBackendValues generates values.yaml for backend services.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - string: values.yaml content
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building and environment resolution
func (r *Renderer) generateBackendValues(name string, service configschema.BackendService, config *configschema.Config) (string, error) {
	var builder strings.Builder

	builder.WriteString("replicaCount: 1\n\n")
	builder.WriteString("image:\n")
	// Use auto-calculated image name
	imageName := config.GetImageName(name)
	builder.WriteString("  repository: " + config.DockerRegistry + "/" + imageName + "\n")
	builder.WriteString("  tag: \"\"\n")
	builder.WriteString("  pullPolicy: IfNotPresent\n\n")

	// Ports
	ports := service.Ports
	if ports == nil {
		ports = &config.BackendDefaults.Ports
	}

	builder.WriteString("service:\n")
	builder.WriteString("  type: ClusterIP\n")
	builder.WriteString("  ports:\n")
	builder.WriteString("    http: " + strconv.Itoa(ports.HTTP) + "\n")
	builder.WriteString("    health: " + strconv.Itoa(ports.Health) + "\n")
	builder.WriteString("    metrics: " + strconv.Itoa(ports.Metrics) + "\n\n")

	// Environment variables
	builder.WriteString("env:\n")

	// Global environment
	for key, value := range config.Env.Global {
		builder.WriteString("  " + key + ": \"" + value + "\"\n")
	}

	// Backend environment
	for key, value := range config.Env.Backend {
		builder.WriteString("  " + key + ": \"" + value + "\"\n")
	}

	// Service-specific environment
	for key, value := range service.Env.Common {
		builder.WriteString("  " + key + ": \"" + value + "\"\n")
	}

	// Kubernetes-specific environment
	for key, value := range service.Env.Kubernetes {
		// Resolve expressions for Kubernetes environment
		resolved, err := r.refParser.ReplaceAll(value, ref.EnvironmentKubernetes, config)
		if err != nil {
			return "", fmt.Errorf("failed to resolve expression %s: %w", value, err)
		}
		builder.WriteString("  " + key + ": \"" + resolved + "\"\n")
	}

	// Port environment variables
	builder.WriteString("  HTTP_PORT: \"" + strconv.Itoa(ports.HTTP) + "\"\n")
	builder.WriteString("  HEALTH_PORT: \"" + strconv.Itoa(ports.Health) + "\"\n")
	builder.WriteString("  METRICS_PORT: \"" + strconv.Itoa(ports.Metrics) + "\"\n")

	return builder.String(), nil
}

// generateFrontendValues generates values.yaml for frontend services.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - string: values.yaml content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) generateFrontendValues(name string, service configschema.FrontendService, config *configschema.Config) string {
	var builder strings.Builder

	builder.WriteString("replicaCount: 1\n\n")
	builder.WriteString("image:\n")
	// Use auto-calculated image name
	imageName := config.GetImageName(name)
	builder.WriteString("  repository: " + config.DockerRegistry + "/" + imageName + "\n")
	builder.WriteString("  tag: \"\"\n")
	builder.WriteString("  pullPolicy: IfNotPresent\n\n")

	builder.WriteString("service:\n")
	builder.WriteString("  type: ClusterIP\n")
	builder.WriteString("  ports:\n")
	builder.WriteString("    http: 3000\n\n")

	// Environment variables
	builder.WriteString("env:\n")

	// Global environment
	for key, value := range config.Env.Global {
		builder.WriteString("  " + key + ": \"" + value + "\"\n")
	}

	// Frontend environment
	for key, value := range config.Env.Frontend {
		builder.WriteString("  " + key + ": \"" + value + "\"\n")
	}

	return builder.String()
}

// generateDatabaseValues generates values.yaml for database service.
//
// Parameters:
//   - db: Database configuration
//   - config: Project configuration
//
// Returns:
//   - string: values.yaml content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) generateDatabaseValues(db configschema.DatabaseConfig, config *configschema.Config) string {
	var builder strings.Builder

	builder.WriteString("replicaCount: 1\n\n")
	builder.WriteString("image:\n")
	builder.WriteString("  repository: mysql\n")
	builder.WriteString("  tag: \"9.4\"\n")
	builder.WriteString("  pullPolicy: IfNotPresent\n\n")

	builder.WriteString("service:\n")
	builder.WriteString("  type: ClusterIP\n")
	builder.WriteString("  ports:\n")
	builder.WriteString("    mysql: " + strconv.Itoa(db.Port) + "\n\n")

	builder.WriteString("database:\n")
	builder.WriteString("  name: \"" + db.Database + "\"\n")
	builder.WriteString("  user: \"" + db.User + "\"\n")
	builder.WriteString("  password: \"" + db.Password + "\"\n")
	builder.WriteString("  rootPassword: \"" + db.RootPassword + "\"\n")

	return builder.String()
}

// generateBackendDeployment generates Deployment template for backend services.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - string: Deployment YAML content
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateBackendDeployment(name string, service configschema.BackendService, config *configschema.Config) (string, error) {
	ports := service.Ports
	if ports == nil {
		ports = &config.BackendDefaults.Ports
	}

	template := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "%s.fullname" . }}
  labels:
    {{- include "%s.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "%s.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "%s.selectorLabels" . | nindent 8 }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.service.ports.http }}
              protocol: TCP
            - name: health
              containerPort: {{ .Values.service.ports.health }}
              protocol: TCP
            - name: metrics
              containerPort: {{ .Values.service.ports.metrics }}
              protocol: TCP
          env:
            {{- range $key, $value := .Values.env }}
            - name: {{ $key }}
              value: {{ $value | quote }}
            {{- end }}
          livenessProbe:
            httpGet:
              path: /health
              port: health
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: health
            initialDelaySeconds: 5
            periodSeconds: 5
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
`

	return fmt.Sprintf(template, name, name, name, name), nil
}

// generateFrontendDeployment generates Deployment template for frontend services.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - string: Deployment YAML content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateFrontendDeployment(name string, service configschema.FrontendService, config *configschema.Config) string {
	template := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "%s.fullname" . }}
  labels:
    {{- include "%s.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "%s.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "%s.selectorLabels" . | nindent 8 }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.service.ports.http }}
              protocol: TCP
          env:
            {{- range $key, $value := .Values.env }}
            - name: {{ $key }}
              value: {{ $value | quote }}
            {{- end }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
`

	return fmt.Sprintf(template, name, name, name, name)
}

// generateDatabaseDeployment generates Deployment template for database service.
//
// Parameters:
//   - db: Database configuration
//   - config: Project configuration
//
// Returns:
//   - string: Deployment YAML content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateDatabaseDeployment(db configschema.DatabaseConfig, config *configschema.Config) string {
	template := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "mysql.fullname" . }}
  labels:
    {{- include "mysql.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "mysql.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "mysql.selectorLabels" . | nindent 8 }}
    spec:
      containers:
        - name: mysql
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: mysql
              containerPort: {{ .Values.service.ports.mysql }}
              protocol: TCP
          env:
            - name: MYSQL_ROOT_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ include "mysql.fullname" . }}
                  key: root-password
            - name: MYSQL_DATABASE
              value: {{ .Values.database.name | quote }}
            - name: MYSQL_USER
              value: {{ .Values.database.user | quote }}
            - name: MYSQL_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ include "mysql.fullname" . }}
                  key: password
          livenessProbe:
            exec:
              command:
                - mysqladmin
                - ping
                - -h
                - localhost
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            exec:
              command:
                - mysqladmin
                - ping
                - -h
                - localhost
            initialDelaySeconds: 5
            periodSeconds: 5
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
`

	return template
}

// generateBackendServices generates Service templates for backend services.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//   - templatesDir: Templates directory path
//
// Returns:
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering and file I/O
func (r *Renderer) generateBackendServices(name string, service configschema.BackendService, config *configschema.Config, templatesDir string) error {
	// Generate clusterIP service
	clusterIPYAML := r.generateClusterIPService(name, service, config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "service-clusterip.yaml"), clusterIPYAML, 0644); err != nil {
		return fmt.Errorf("failed to write service-clusterip.yaml: %w", err)
	}

	// Generate headless service
	headlessYAML := r.generateHeadlessService(name, service, config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "service-headless.yaml"), headlessYAML, 0644); err != nil {
		return fmt.Errorf("failed to write service-headless.yaml: %w", err)
	}

	return nil
}

// generateClusterIPService generates ClusterIP Service template.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - string: Service YAML content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateClusterIPService(name string, service configschema.BackendService, config *configschema.Config) string {
	template := `apiVersion: v1
kind: Service
metadata:
  name: {{ include "%s.fullname" . }}
  labels:
    {{- include "%s.labels" . | nindent 4 }}
spec:
  type: ClusterIP
  ports:
    - port: {{ .Values.service.ports.http }}
      targetPort: http
      protocol: TCP
      name: http
    - port: {{ .Values.service.ports.health }}
      targetPort: health
      protocol: TCP
      name: health
    - port: {{ .Values.service.ports.metrics }}
      targetPort: metrics
      protocol: TCP
      name: metrics
  selector:
    {{- include "%s.selectorLabels" . | nindent 4 }}
`

	return fmt.Sprintf(template, name, name, name)
}

// generateHeadlessService generates Headless Service template.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - string: Service YAML content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateHeadlessService(name string, service configschema.BackendService, config *configschema.Config) string {
	template := `apiVersion: v1
kind: Service
metadata:
  name: {{ include "%s.fullname" . }}-headless
  labels:
    {{- include "%s.labels" . | nindent 4 }}
spec:
  type: ClusterIP
  clusterIP: None
  publishNotReadyAddresses: true
  ports:
    - port: {{ .Values.service.ports.http }}
      targetPort: http
      protocol: TCP
      name: http
    - port: {{ .Values.service.ports.health }}
      targetPort: health
      protocol: TCP
      name: health
    - port: {{ .Values.service.ports.metrics }}
      targetPort: metrics
      protocol: TCP
      name: metrics
  selector:
    {{- include "%s.selectorLabels" . | nindent 4 }}
`

	return fmt.Sprintf(template, name, name, name)
}

// generateFrontendService generates Service template for frontend services.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - string: Service YAML content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateFrontendService(name string, service configschema.FrontendService, config *configschema.Config) string {
	template := `apiVersion: v1
kind: Service
metadata:
  name: {{ include "%s.fullname" . }}
  labels:
    {{- include "%s.labels" . | nindent 4 }}
spec:
  type: ClusterIP
  ports:
    - port: {{ .Values.service.ports.http }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    {{- include "%s.selectorLabels" . | nindent 4 }}
`

	return fmt.Sprintf(template, name, name, name)
}

// generateDatabaseService generates Service template for database service.
//
// Parameters:
//   - db: Database configuration
//   - config: Project configuration
//
// Returns:
//   - string: Service YAML content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateDatabaseService(db configschema.DatabaseConfig, config *configschema.Config) string {
	template := `apiVersion: v1
kind: Service
metadata:
  name: {{ include "mysql.fullname" . }}
  labels:
    {{- include "mysql.labels" . | nindent 4 }}
spec:
  type: ClusterIP
  ports:
    - port: {{ .Values.service.ports.mysql }}
      targetPort: mysql
      protocol: TCP
      name: mysql
  selector:
    {{- include "mysql.selectorLabels" . | nindent 4 }}
`

	return template
}

// generateBackendConfigMap generates ConfigMap template for backend services.
//
// Parameters:
//   - name: Service name
//   - service: Service configuration
//   - config: Project configuration
//
// Returns:
//   - string: ConfigMap YAML content
//   - error: Generation error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateBackendConfigMap(name string, service configschema.BackendService, config *configschema.Config) (string, error) {
	template := `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "%s.fullname" . }}-config
  labels:
    {{- include "%s.labels" . | nindent 4 }}
data:
  {{- range $key, $value := .Values.env }}
  {{ $key }}: {{ $value | quote }}
  {{- end }}
`

	return fmt.Sprintf(template, name, name), nil
}

// generateDatabaseSecret generates Secret template for database service.
//
// Parameters:
//   - db: Database configuration
//   - config: Project configuration
//
// Returns:
//   - string: Secret YAML content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateDatabaseSecret(db configschema.DatabaseConfig, config *configschema.Config) string {
	template := `apiVersion: v1
kind: Secret
metadata:
  name: {{ include "mysql.fullname" . }}
  labels:
    {{- include "mysql.labels" . | nindent 4 }}
type: Opaque
data:
  root-password: {{ .Values.database.rootPassword | b64enc }}
  password: {{ .Values.database.password | b64enc }}
`

	return template
}

// generateConfigMap generates ConfigMap template.
//
// Parameters:
//   - name: ConfigMap name
//   - data: ConfigMap data
//   - config: Project configuration
//
// Returns:
//   - string: ConfigMap YAML content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateConfigMap(name string, data map[string]string, config *configschema.Config) string {
	var builder strings.Builder

	builder.WriteString("apiVersion: v1\n")
	builder.WriteString("kind: ConfigMap\n")
	builder.WriteString("metadata:\n")
	builder.WriteString("  name: " + name + "\n")
	builder.WriteString("  labels:\n")
	builder.WriteString("    app.kubernetes.io/name: resources\n")
	builder.WriteString("    app.kubernetes.io/instance: " + config.ProjectName + "\n")
	builder.WriteString("data:\n")

	for key, value := range data {
		builder.WriteString("  " + key + ": \"" + value + "\"\n")
	}

	return builder.String()
}

// generateSecret generates Secret template.
//
// Parameters:
//   - name: Secret name
//   - data: Secret data
//   - config: Project configuration
//
// Returns:
//   - string: Secret YAML content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - Template rendering
func (r *Renderer) generateSecret(name string, data map[string]string, config *configschema.Config) string {
	var builder strings.Builder

	builder.WriteString("apiVersion: v1\n")
	builder.WriteString("kind: Secret\n")
	builder.WriteString("metadata:\n")
	builder.WriteString("  name: " + name + "\n")
	builder.WriteString("  labels:\n")
	builder.WriteString("    app.kubernetes.io/name: resources\n")
	builder.WriteString("    app.kubernetes.io/instance: " + config.ProjectName + "\n")
	builder.WriteString("type: Opaque\n")
	builder.WriteString("data:\n")

	for key, value := range data {
		builder.WriteString("  " + key + ": " + value + "\n")
	}

	return builder.String()
}
