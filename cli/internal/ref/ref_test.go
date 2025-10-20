package ref

import (
	"testing"
)

func TestParseExpression(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name        string
		input       string
		expectError bool
		exprType    ExpressionType
		resource    string
		key         string
		serviceType string
	}{
		{
			name:        "ConfigMap reference",
			input:       "${cfg:global-config}",
			expectError: false,
			exprType:    TypeConfigMap,
			resource:    "global-config",
		},
		{
			name:        "ConfigMap value reference",
			input:       "${cfgv:global-config:KEY}",
			expectError: false,
			exprType:    TypeConfigMapValue,
			resource:    "global-config",
			key:         "KEY",
		},
		{
			name:        "Secret reference",
			input:       "${sec:jwt-secret:TOKEN}",
			expectError: false,
			exprType:    TypeSecret,
			resource:    "jwt-secret",
			key:         "TOKEN",
		},
		{
			name:        "Service reference with type",
			input:       "${svc:user-service@headless}",
			expectError: false,
			exprType:    TypeService,
			resource:    "user-service",
			serviceType: "headless",
		},
		{
			name:        "Invalid expression",
			input:       "not-an-expression",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.Parse(tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if expr.Type != tt.exprType {
				t.Errorf("Expected type %s, got %s", tt.exprType, expr.Type)
			}

			if expr.Resource != tt.resource {
				t.Errorf("Expected resource '%s', got '%s'", tt.resource, expr.Resource)
			}

			if tt.key != "" && expr.Key != tt.key {
				t.Errorf("Expected key '%s', got '%s'", tt.key, expr.Key)
			}

			if tt.serviceType != "" && expr.ServiceType != tt.serviceType {
				t.Errorf("Expected service type '%s', got '%s'", tt.serviceType, expr.ServiceType)
			}
		})
	}
}

func TestParseAll(t *testing.T) {
	parser := NewParser()

	text := "Use ${cfg:config1} and ${cfgv:config2:KEY} with ${sec:secret1:TOKEN}"

	expressions, err := parser.ParseAll(text)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(expressions) != 3 {
		t.Errorf("Expected 3 expressions, got %d", len(expressions))
	}

	expectedTypes := []ExpressionType{TypeConfigMap, TypeConfigMapValue, TypeSecret}
	for i, expr := range expressions {
		if expr.Type != expectedTypes[i] {
			t.Errorf("Expected expression %d to be %s, got %s", i, expectedTypes[i], expr.Type)
		}
	}
}

func TestResolveConfigMap(t *testing.T) {
	parser := NewParser()

	expr := &Expression{
		Type:     TypeConfigMap,
		Resource: "global-config",
	}

	// Test Kubernetes environment
	result, err := parser.Resolve(expr, EnvironmentKubernetes, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result.Value != "global-config" {
		t.Errorf("Expected value 'global-config', got '%s'", result.Value)
	}

	// Test Compose environment (should fail)
	_, err = parser.Resolve(expr, EnvironmentCompose, nil)
	if err == nil {
		t.Error("Expected error for ConfigMap in Compose environment")
	}
}

func TestValidationHelpers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(string) bool
		expected bool
	}{
		{"valid resource name", "my-service", isValidResourceName, true},
		{"invalid resource name with uppercase", "My-Service", isValidResourceName, false},
		{"valid key name", "MY_KEY", isValidKeyName, true},
		{"invalid key name with space", "MY KEY", isValidKeyName, false},
		{"valid service type", "headless", isValidServiceType, true},
		{"invalid service type", "unknown", isValidServiceType, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.validate(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
