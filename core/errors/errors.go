// Package errors provides structured error handling compatible with standard library.
//
// Overview:
//   - Responsibility: Define error codes and structured error wrapping
//   - Key Types: Code type for error classification, E struct for structured errors
//   - Concurrency Model: All functions are safe for concurrent use
//   - Error Semantics: Compatible with standard library error wrapping
//   - Performance Notes: Minimal allocations, designed for high-throughput scenarios
//
// Usage:
//
//	err := errors.New(errors.CodeInvalidArgument, "invalid user ID")
//	wrapped := errors.Wrap(errors.CodeInternal, "user service", originalErr)
//	code := errors.CodeOf(err)
package errors

import (
	"errors"
	"fmt"
)

// Code represents an error classification code.
// Common codes include "INVALID_ARGUMENT", "NOT_FOUND", "INTERNAL", etc.
type Code string

// Common error codes
const (
	CodeInvalidArgument  Code = "INVALID_ARGUMENT"
	CodeNotFound         Code = "NOT_FOUND"
	CodeAlreadyExists    Code = "ALREADY_EXISTS"
	CodePermissionDenied Code = "PERMISSION_DENIED"
	CodeUnauthenticated  Code = "UNAUTHENTICATED"
	CodeInternal         Code = "INTERNAL"
	CodeUnavailable      Code = "UNAVAILABLE"
	CodeDeadlineExceeded Code = "DEADLINE_EXCEEDED"
)

// E represents a structured error with code, operation, and message.
type E struct {
	Code Code   // Error classification code
	Op   string // Operation that failed
	Err  error  // Underlying error (may be nil)
	Msg  string // Human-readable message
}

// Error implements the error interface.
func (e *E) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Msg, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Msg)
}

// Unwrap returns the underlying error for error unwrapping.
func (e *E) Unwrap() error {
	return e.Err
}

// New creates a new structured error with the given code and message.
func New(code Code, msg string) error {
	return &E{
		Code: code,
		Msg:  msg,
	}
}

// Wrap creates a new structured error wrapping an existing error.
// The operation name helps identify where the error occurred.
func Wrap(code Code, op string, err error) error {
	return &E{
		Code: code,
		Op:   op,
		Err:  err,
		Msg:  "",
	}
}

// Wrapf creates a new structured error wrapping an existing error with formatted message.
func Wrapf(code Code, op string, err error, format string, args ...any) error {
	return &E{
		Code: code,
		Op:   op,
		Err:  err,
		Msg:  fmt.Sprintf(format, args...),
	}
}

// CodeOf extracts the error code from an error.
// Returns empty string if the error doesn't have a code.
func CodeOf(err error) Code {
	var e *E
	if err != nil && As(err, &e) {
		return e.Code
	}
	return ""
}

// As is a type assertion helper for error unwrapping.
// This is a convenience function that works with the standard library's errors.As.
func As(err error, target **E) bool {
	return errors.As(err, target)
}
