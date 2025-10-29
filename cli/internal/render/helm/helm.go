// Package helm provides Helm chart rendering for egg projects.
//
// Overview:
//   - Responsibility: Render unified Helm chart from egg configuration
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

// Render renders a unified Helm chart for the entire project.
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
	ui.Info("Rendering unified Helm chart for project: %s", config.ProjectName)

	// Create project-level chart directory
	chartDir := filepath.Join("deploy/helm", config.ProjectName)
	if err := r.fs.CreateDirectory(chartDir); err != nil {
		return fmt.Errorf("failed to create chart directory: %w", err)
	}

	// Create templates directory
	templatesDir := filepath.Join(chartDir, "templates")
	if err := r.fs.CreateDirectory(templatesDir); err != nil {
		return fmt.Errorf("failed to create templates directory: %w", err)
	}

	// Generate Chart.yaml
	chartYAML := r.generateProjectChart(config)
	if err := r.fs.WriteFile(filepath.Join(chartDir, "Chart.yaml"), chartYAML, 0644); err != nil {
		return fmt.Errorf("failed to write Chart.yaml: %w", err)
	}

	// Generate values.yaml
	valuesYAML, err := r.generateProjectValues(config)
	if err != nil {
		return fmt.Errorf("failed to generate values.yaml: %w", err)
	}
	if err := r.fs.WriteFile(filepath.Join(chartDir, "values.yaml"), valuesYAML, 0644); err != nil {
		return fmt.Errorf("failed to write values.yaml: %w", err)
	}

	// Generate templates
	if err := r.generateUnifiedTemplates(config, templatesDir); err != nil {
		return fmt.Errorf("failed to generate templates: %w", err)
	}

	ui.Success("Helm chart rendered: %s", chartDir)
	return nil
}

// generateProjectChart generates Chart.yaml for the project.
//
// Parameters:
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
func (r *Renderer) generateProjectChart(config *configschema.Config) string {
	return fmt.Sprintf(`apiVersion: v2
name: %s
description: Helm chart for %s
type: application
version: 0.1.0
appVersion: "%s"
`, config.ProjectName, config.ProjectName, config.Version)
}

// generateProjectValues generates values.yaml containing all services.
//
// Parameters:
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
func (r *Renderer) generateProjectValues(config *configschema.Config) (string, error) {
	var builder strings.Builder

	// Project metadata
	builder.WriteString(fmt.Sprintf("projectName: %s\n", config.ProjectName))
	builder.WriteString(fmt.Sprintf("dockerRegistry: %s\n", config.DockerRegistry))
	builder.WriteString(fmt.Sprintf("version: %s\n\n", config.Version))

	// Backend services
	builder.WriteString("backend:\n")
	for name, service := range config.Backend {
		builder.WriteString(fmt.Sprintf("  %s:\n", name))
		builder.WriteString("    enabled: true\n")

		// Image name
		imageName := config.GetImageName(name)
		builder.WriteString(fmt.Sprintf("    image: %s/%s:%s\n", config.DockerRegistry, imageName, config.Version))

		// Replicas
		builder.WriteString("    replicas: 2\n")

		// Ports
		ports := service.Ports
		if ports == nil {
			ports = &config.BackendDefaults.Ports
		}
		builder.WriteString("    ports:\n")
		builder.WriteString(fmt.Sprintf("      http: %d\n", ports.HTTP))
		builder.WriteString(fmt.Sprintf("      health: %d\n", ports.Health))
		builder.WriteString(fmt.Sprintf("      metrics: %d\n", ports.Metrics))

		// Environment variables
		builder.WriteString("    env:\n")

		// Global environment
		for key, value := range config.Env.Global {
			builder.WriteString(fmt.Sprintf("      - name: %s\n", key))
			builder.WriteString(fmt.Sprintf("        value: \"%s\"\n", value))
		}

		// Backend environment
		for key, value := range config.Env.Backend {
			builder.WriteString(fmt.Sprintf("      - name: %s\n", key))
			builder.WriteString(fmt.Sprintf("        value: \"%s\"\n", value))
		}

		// Service-specific environment
		for key, value := range service.Env.Common {
			builder.WriteString(fmt.Sprintf("      - name: %s\n", key))
			builder.WriteString(fmt.Sprintf("        value: \"%s\"\n", value))
		}

		// Kubernetes-specific environment (with expression resolution)
		for key, value := range service.Env.Kubernetes {
			resolved, err := r.refParser.ReplaceAll(value, ref.EnvironmentKubernetes, config)
			if err != nil {
				return "", fmt.Errorf("failed to resolve expression %s: %w", value, err)
			}
			builder.WriteString(fmt.Sprintf("      - name: %s\n", key))
			builder.WriteString(fmt.Sprintf("        value: \"%s\"\n", resolved))
		}

		// Port environment variables
		builder.WriteString("      - name: HTTP_PORT\n")
		builder.WriteString(fmt.Sprintf("        value: \"%d\"\n", ports.HTTP))
		builder.WriteString("      - name: HEALTH_PORT\n")
		builder.WriteString(fmt.Sprintf("        value: \"%d\"\n", ports.Health))
		builder.WriteString("      - name: METRICS_PORT\n")
		builder.WriteString(fmt.Sprintf("        value: \"%d\"\n", ports.Metrics))

		// Resources
		builder.WriteString("    resources:\n")
		builder.WriteString("      requests:\n")
		builder.WriteString("        cpu: 100m\n")
		builder.WriteString("        memory: 128Mi\n")
		builder.WriteString("      limits:\n")
		builder.WriteString("        cpu: 500m\n")
		builder.WriteString("        memory: 512Mi\n")
	}

	// Frontend services
	builder.WriteString("\nfrontend:\n")
	for name := range config.Frontend {
		builder.WriteString(fmt.Sprintf("  %s:\n", name))
		builder.WriteString("    enabled: true\n")

		// Image name
		imageName := config.GetImageName(name)
		builder.WriteString(fmt.Sprintf("    image: %s/%s:%s\n", config.DockerRegistry, imageName, config.Version))

		// Replicas
		builder.WriteString("    replicas: 1\n")

		// Ports
		builder.WriteString("    ports:\n")
		builder.WriteString("      http: 3000\n")

		// Environment variables
		builder.WriteString("    env:\n")

		// Global environment
		for key, value := range config.Env.Global {
			builder.WriteString(fmt.Sprintf("      - name: %s\n", key))
			builder.WriteString(fmt.Sprintf("        value: \"%s\"\n", value))
		}

		// Frontend environment
		for key, value := range config.Env.Frontend {
			builder.WriteString(fmt.Sprintf("      - name: %s\n", key))
			builder.WriteString(fmt.Sprintf("        value: \"%s\"\n", value))
		}

		// Resources
		builder.WriteString("    resources:\n")
		builder.WriteString("      requests:\n")
		builder.WriteString("        cpu: 50m\n")
		builder.WriteString("        memory: 64Mi\n")
		builder.WriteString("      limits:\n")
		builder.WriteString("        cpu: 200m\n")
		builder.WriteString("        memory: 256Mi\n")
	}

	// Global ConfigMaps
	builder.WriteString("\nglobalConfigMaps:\n")
	for name, data := range config.Kubernetes.Resources.ConfigMaps {
		builder.WriteString(fmt.Sprintf("  %s:\n", name))
		for key, value := range data {
			builder.WriteString(fmt.Sprintf("    %s: \"%s\"\n", key, value))
		}
	}

	// Global Secrets
	builder.WriteString("\nglobalSecrets:\n")
	for name, data := range config.Kubernetes.Resources.Secrets {
		builder.WriteString(fmt.Sprintf("  %s:\n", name))
		for key, value := range data {
			builder.WriteString(fmt.Sprintf("    %s: %s\n", key, value))
		}
	}

	return builder.String(), nil
}

// generateUnifiedTemplates generates unified Helm templates.
//
// Parameters:
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
func (r *Renderer) generateUnifiedTemplates(config *configschema.Config, templatesDir string) error {
	// Generate _helpers.tpl
	helpersTPL := r.generateHelpersTPL(config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "_helpers.tpl"), helpersTPL, 0644); err != nil {
		return fmt.Errorf("failed to write _helpers.tpl: %w", err)
	}

	// Generate backend deployment template
	backendDeploymentTPL := r.generateBackendDeploymentTPL(config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "backend-deployment.yaml"), backendDeploymentTPL, 0644); err != nil {
		return fmt.Errorf("failed to write backend-deployment.yaml: %w", err)
	}

	// Generate backend service template
	backendServiceTPL := r.generateBackendServiceTPL(config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "backend-service.yaml"), backendServiceTPL, 0644); err != nil {
		return fmt.Errorf("failed to write backend-service.yaml: %w", err)
	}

	// Generate frontend deployment template
	frontendDeploymentTPL := r.generateFrontendDeploymentTPL(config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "frontend-deployment.yaml"), frontendDeploymentTPL, 0644); err != nil {
		return fmt.Errorf("failed to write frontend-deployment.yaml: %w", err)
	}

	// Generate frontend service template
	frontendServiceTPL := r.generateFrontendServiceTPL(config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "frontend-service.yaml"), frontendServiceTPL, 0644); err != nil {
		return fmt.Errorf("failed to write frontend-service.yaml: %w", err)
	}

	// Generate ConfigMaps template
	configMapsTPL := r.generateConfigMapsTPL(config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "configmaps.yaml"), configMapsTPL, 0644); err != nil {
		return fmt.Errorf("failed to write configmaps.yaml: %w", err)
	}

	// Generate Secrets template
	secretsTPL := r.generateSecretsTPL(config)
	if err := r.fs.WriteFile(filepath.Join(templatesDir, "secrets.yaml"), secretsTPL, 0644); err != nil {
		return fmt.Errorf("failed to write secrets.yaml: %w", err)
	}

	return nil
}

// generateHelpersTPL generates _helpers.tpl.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - string: Template content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) generateHelpersTPL(config *configschema.Config) string {
	projectName := config.ProjectName
	return fmt.Sprintf(`{{/*
Expand the name of the chart.
*/}}
{{- define "%s.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "%s.fullname" -}}
{{- if .Values.nameOverride }}
{{- .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%%s-%%s" .Chart.Name .Release.Name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "%s.chart" -}}
{{- printf "%%s-%%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "%s.labels" -}}
helm.sh/chart: {{ include "%s.chart" . }}
{{ include "%s.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "%s.selectorLabels" -}}
app.kubernetes.io/name: {{ include "%s.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
`, projectName, projectName, projectName, projectName, projectName, projectName, projectName, projectName)
}

// generateBackendDeploymentTPL generates backend deployment template.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - string: Template content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) generateBackendDeploymentTPL(config *configschema.Config) string {
	templateName := config.ProjectName
	return fmt.Sprintf(`{{- range $name, $service := .Values.backend }}
{{- if $service.enabled }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "%s.fullname" $ }}-{{ $name }}
  labels:
    {{- include "%s.labels" $ | nindent 4 }}
    app.kubernetes.io/component: backend
    app.kubernetes.io/name: {{ $name }}
spec:
  replicas: {{ $service.replicas | default 2 }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ $name }}
      app.kubernetes.io/instance: {{ $.Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ $name }}
        app.kubernetes.io/instance: {{ $.Release.Name }}
    spec:
      containers:
      - name: {{ $name }}
        image: {{ $service.image }}
        imagePullPolicy: IfNotPresent
        ports:
        - name: http
          containerPort: {{ $service.ports.http }}
          protocol: TCP
        - name: health
          containerPort: {{ $service.ports.health }}
          protocol: TCP
        - name: metrics
          containerPort: {{ $service.ports.metrics }}
          protocol: TCP
        env:
        {{- range $service.env }}
        - name: {{ .name }}
          value: {{ .value | quote }}
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
          {{- toYaml $service.resources | nindent 10 }}
{{- end }}
{{- end }}
`, templateName, templateName)
}

// generateBackendServiceTPL generates backend service template.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - string: Template content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) generateBackendServiceTPL(config *configschema.Config) string {
	templateName := config.ProjectName
	return fmt.Sprintf(`{{- range $name, $service := .Values.backend }}
{{- if $service.enabled }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ include "%s.fullname" $ }}-{{ $name }}
  labels:
    {{- include "%s.labels" $ | nindent 4 }}
    app.kubernetes.io/component: backend
    app.kubernetes.io/name: {{ $name }}
spec:
  type: ClusterIP
  ports:
  - port: {{ $service.ports.http }}
    targetPort: http
    protocol: TCP
    name: http
  - port: {{ $service.ports.health }}
    targetPort: health
    protocol: TCP
    name: health
  - port: {{ $service.ports.metrics }}
    targetPort: metrics
    protocol: TCP
    name: metrics
  selector:
    app.kubernetes.io/name: {{ $name }}
    app.kubernetes.io/instance: {{ $.Release.Name }}
{{- end }}
{{- end }}
`, templateName, templateName)
}

// generateFrontendDeploymentTPL generates frontend deployment template.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - string: Template content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) generateFrontendDeploymentTPL(config *configschema.Config) string {
	templateName := config.ProjectName
	return fmt.Sprintf(`{{- range $name, $service := .Values.frontend }}
{{- if $service.enabled }}
{{- $safeName := $name | replace "_" "-" | lower }}
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "%s.fullname" $ }}-{{ $safeName }}
  labels:
    {{- include "%s.labels" $ | nindent 4 }}
    app.kubernetes.io/component: frontend
    app.kubernetes.io/name: {{ $name }}
spec:
  replicas: {{ $service.replicas | default 1 }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ $name }}
      app.kubernetes.io/instance: {{ $.Release.Name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ $name }}
        app.kubernetes.io/instance: {{ $.Release.Name }}
    spec:
      containers:
      - name: {{ $name }}
        image: {{ $service.image }}
        imagePullPolicy: IfNotPresent
        ports:
        - name: http
          containerPort: {{ $service.ports.http }}
          protocol: TCP
        env:
        {{- range $service.env }}
        - name: {{ .name }}
          value: {{ .value | quote }}
        {{- end }}
        resources:
          {{- toYaml $service.resources | nindent 10 }}
{{- end }}
{{- end }}
`, templateName, templateName)
}

// generateFrontendServiceTPL generates frontend service template.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - string: Template content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) generateFrontendServiceTPL(config *configschema.Config) string {
	templateName := config.ProjectName
	return fmt.Sprintf(`{{- range $name, $service := .Values.frontend }}
{{- if $service.enabled }}
{{- $safeName := $name | replace "_" "-" | lower }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ include "%s.fullname" $ }}-{{ $safeName }}
  labels:
    {{- include "%s.labels" $ | nindent 4 }}
    app.kubernetes.io/component: frontend
    app.kubernetes.io/name: {{ $name }}
spec:
  type: ClusterIP
  ports:
  - port: {{ $service.ports.http }}
    targetPort: http
    protocol: TCP
    name: http
  selector:
    app.kubernetes.io/name: {{ $name }}
    app.kubernetes.io/instance: {{ $.Release.Name }}
{{- end }}
{{- end }}
`, templateName, templateName)
}

// generateConfigMapsTPL generates ConfigMaps template.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - string: Template content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) generateConfigMapsTPL(config *configschema.Config) string {
	templateName := config.ProjectName
	return fmt.Sprintf(`{{- range $name, $data := .Values.globalConfigMaps }}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ $name }}
  labels:
    {{- include "%s.labels" $ | nindent 4 }}
data:
  {{- range $key, $value := $data }}
  {{ $key }}: {{ $value | quote }}
  {{- end }}
{{- end }}
`, templateName)
}

// generateSecretsTPL generates Secrets template.
//
// Parameters:
//   - config: Project configuration
//
// Returns:
//   - string: Template content
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - String building
func (r *Renderer) generateSecretsTPL(config *configschema.Config) string {
	templateName := config.ProjectName
	return fmt.Sprintf(`{{- range $name, $data := .Values.globalSecrets }}
---
apiVersion: v1
kind: Secret
metadata:
  name: {{ $name }}
  labels:
    {{- include "%s.labels" $ | nindent 4 }}
type: Opaque
data:
  {{- range $key, $value := $data }}
  {{ $key }}: {{ $value }}
  {{- end }}
{{- end }}
`, templateName)
}
