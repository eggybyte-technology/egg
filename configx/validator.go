// Package configx provides configuration validation.
package configx

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// ValidatorOption configures the validator.
type ValidatorOption func(*validator.Validate)

// NewValidator creates a new validator instance.
func NewValidator(opts ...ValidatorOption) *validator.Validate {
	v := validator.New()
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// ValidateStruct validates a struct using validator tags.
func ValidateStruct(v *validator.Validate, target any) error {
	if v == nil {
		v = validator.New()
	}

	if err := v.Struct(target); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}
