package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeInvalidArgument, "test error")
	if err == nil {
		t.Fatal("New should return non-nil error")
	}

	if err.Error() == "" {
		t.Fatal("Error message should not be empty")
	}

	// Check if it's our custom error type
	var customErr *E
	if !errors.As(err, &customErr) {
		t.Fatal("Error should be of type *E")
	}

	if customErr.Code != CodeInvalidArgument {
		t.Errorf("Expected code %s, got %s", CodeInvalidArgument, customErr.Code)
	}

	if customErr.Msg != "test error" {
		t.Errorf("Expected message %q, got %q", "test error", customErr.Msg)
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := Wrap(CodeInternal, "operation", originalErr)

	if wrappedErr == nil {
		t.Fatal("Wrap should return non-nil error")
	}

	// Check if it's our custom error type
	var customErr *E
	if !errors.As(wrappedErr, &customErr) {
		t.Fatal("Wrapped error should be of type *E")
	}

	if customErr.Code != CodeInternal {
		t.Errorf("Expected code %s, got %s", CodeInternal, customErr.Code)
	}

	if customErr.Op != "operation" {
		t.Errorf("Expected operation %q, got %q", "operation", customErr.Op)
	}

	if customErr.Err != originalErr {
		t.Error("Wrapped error should contain original error")
	}
}

func TestWrapf(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := Wrapf(CodeNotFound, "operation", originalErr, "formatted message: %s", "test")

	if wrappedErr == nil {
		t.Fatal("Wrapf should return non-nil error")
	}

	// Check if it's our custom error type
	var customErr *E
	if !errors.As(wrappedErr, &customErr) {
		t.Fatal("Wrapped error should be of type *E")
	}

	if customErr.Code != CodeNotFound {
		t.Errorf("Expected code %s, got %s", CodeNotFound, customErr.Code)
	}

	if customErr.Op != "operation" {
		t.Errorf("Expected operation %q, got %q", "operation", customErr.Op)
	}

	if customErr.Msg != "formatted message: test" {
		t.Errorf("Expected message %q, got %q", "formatted message: test", customErr.Msg)
	}
}

func TestCodeOf(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected Code
	}{
		{
			name:     "custom error with code",
			err:      New(CodeInvalidArgument, "test"),
			expected: CodeInvalidArgument,
		},
		{
			name:     "wrapped error with code",
			err:      Wrap(CodeNotFound, "op", errors.New("test")),
			expected: CodeNotFound,
		},
		{
			name:     "standard error",
			err:      errors.New("standard error"),
			expected: "",
		},
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := CodeOf(tt.err)
			if code != tt.expected {
				t.Errorf("Expected code %q, got %q", tt.expected, code)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := Wrap(CodeInternal, "operation", originalErr)

	unwrapped := errors.Unwrap(wrappedErr)
	if unwrapped != originalErr {
		t.Error("Unwrap should return original error")
	}
}

func TestErrorCodes(t *testing.T) {
	codes := []Code{
		CodeInvalidArgument,
		CodeNotFound,
		CodeAlreadyExists,
		CodePermissionDenied,
		CodeUnauthenticated,
		CodeResourceExhausted,
		CodeInternal,
		CodeUnavailable,
		CodeDeadlineExceeded,
		CodeUnimplemented,
		CodeAborted,
		CodeOutOfRange,
		CodeDataLoss,
	}

	for _, code := range codes {
		if code == "" {
			t.Errorf("Error code should not be empty")
		}
	}
}

func TestBuilder(t *testing.T) {
	originalErr := errors.New("database error")
	err := Build(CodeInternal).
		WithOp("user.Create").
		WithErr(originalErr).
		WithMsg("failed to create user").
		WithDetails("user_id", "123", "email", "test@example.com").
		Err()

	if err == nil {
		t.Fatal("Builder should return non-nil error")
	}

	var customErr *E
	if !errors.As(err, &customErr) {
		t.Fatal("Error should be of type *E")
	}

	if customErr.Code != CodeInternal {
		t.Errorf("Expected code %s, got %s", CodeInternal, customErr.Code)
	}

	if customErr.Op != "user.Create" {
		t.Errorf("Expected op %q, got %q", "user.Create", customErr.Op)
	}

	if customErr.Msg != "failed to create user" {
		t.Errorf("Expected msg %q, got %q", "failed to create user", customErr.Msg)
	}

	if customErr.Err != originalErr {
		t.Error("Builder should wrap original error")
	}

	if len(customErr.Details) != 4 {
		t.Errorf("Expected 4 details, got %d", len(customErr.Details))
	}
}

func TestBuilderWithMsgf(t *testing.T) {
	err := Build(CodeNotFound).
		WithOp("user.Get").
		WithMsgf("user %s not found", "u-123").
		Err()

	if err == nil {
		t.Fatal("Builder should return non-nil error")
	}

	var customErr *E
	if !errors.As(err, &customErr) {
		t.Fatal("Error should be of type *E")
	}

	if customErr.Msg != "user u-123 not found" {
		t.Errorf("Expected formatted message, got %q", customErr.Msg)
	}
}
