// Package internal provides tests for configx internal validator implementation.
package internal

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator() should return non-nil validator")
	}
}

func TestNewValidator_WithOptions(t *testing.T) {
	called := false
	opt := func(v *validator.Validate) {
		called = true
	}

	v := NewValidator(opt)
	if v == nil {
		t.Fatal("NewValidator() should return non-nil validator")
	}
	if !called {
		t.Error("Validator option should be called")
	}
}

func TestValidateStruct_Success(t *testing.T) {
	type TestStruct struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	v := validator.New()
	target := TestStruct{
		Name:  "test",
		Email: "test@example.com",
	}

	err := ValidateStruct(v, target)
	if err != nil {
		t.Errorf("ValidateStruct() error = %v, want nil", err)
	}
}

func TestValidateStruct_ValidationError(t *testing.T) {
	type TestStruct struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	v := validator.New()
	target := TestStruct{
		Name:  "", // Required field missing
		Email: "invalid-email", // Invalid email format
	}

	err := ValidateStruct(v, target)
	if err == nil {
		t.Fatal("ValidateStruct() should return error for invalid struct")
	}
	if !contains(err.Error(), "validation failed") {
		t.Errorf("Error message = %q, want to contain 'validation failed'", err.Error())
	}
}

func TestValidateStruct_NilValidator(t *testing.T) {
	type TestStruct struct {
		Name string `validate:"required"`
	}

	target := TestStruct{
		Name: "test",
	}

	// Should create a new validator when nil is passed
	err := ValidateStruct(nil, target)
	if err != nil {
		t.Errorf("ValidateStruct() error = %v, want nil", err)
	}
}

func TestValidateStruct_NilTarget(t *testing.T) {
	v := validator.New()

	err := ValidateStruct(v, nil)
	if err == nil {
		t.Fatal("ValidateStruct() should return error for nil target")
	}
}

func TestValidateStruct_NoValidationTags(t *testing.T) {
	type TestStruct struct {
		Name  string
		Email string
	}

	v := validator.New()
	target := TestStruct{
		Name:  "test",
		Email: "test@example.com",
	}

	err := ValidateStruct(v, target)
	if err != nil {
		t.Errorf("ValidateStruct() error = %v, want nil", err)
	}
}

func TestValidateStruct_MultipleValidators(t *testing.T) {
	type TestStruct struct {
		Name  string `validate:"required,min=3"`
		Email string `validate:"required,email"`
	}

	v := validator.New()

	tests := []struct {
		name    string
		target  TestStruct
		wantErr bool
	}{
		{
			name: "valid",
			target: TestStruct{
				Name:  "test",
				Email: "test@example.com",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			target: TestStruct{
				Name:  "",
				Email: "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "short name",
			target: TestStruct{
				Name:  "ab",
				Email: "test@example.com",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			target: TestStruct{
				Name:  "test",
				Email: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(v, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

