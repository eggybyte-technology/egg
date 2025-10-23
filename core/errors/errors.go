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

// Common error codes (aligned with Connect/gRPC codes)
const (
	CodeInvalidArgument   Code = "INVALID_ARGUMENT"
	CodeNotFound          Code = "NOT_FOUND"
	CodeAlreadyExists     Code = "ALREADY_EXISTS"
	CodePermissionDenied  Code = "PERMISSION_DENIED"
	CodeUnauthenticated   Code = "UNAUTHENTICATED"
	CodeResourceExhausted Code = "RESOURCE_EXHAUSTED"
	CodeInternal          Code = "INTERNAL"
	CodeUnavailable       Code = "UNAVAILABLE"
	CodeDeadlineExceeded  Code = "DEADLINE_EXCEEDED"
	CodeUnimplemented     Code = "UNIMPLEMENTED"
	CodeAborted           Code = "ABORTED"
	CodeOutOfRange        Code = "OUT_OF_RANGE"
	CodeDataLoss          Code = "DATA_LOSS"
)

// E represents a structured error with code, operation, message, and details.
type E struct {
	Code    Code   // Error classification code
	Op      string // Operation that failed
	Err     error  // Underlying error (may be nil)
	Msg     string // Human-readable message
	Details []any  // Additional structured details (e.g., field errors, metadata)
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
	if err != nil && errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// As is a type assertion helper for error unwrapping.
// This is a convenience function that works with the standard library's errors.As.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Is checks if an error has a specific code.
// This is a convenience function for error code checking.
func IsCode(err error, code Code) bool {
	return CodeOf(err) == code
}

// Is checks if an error is of a specific type.
// This is a convenience function for error type checking.
func Is(err error, target error) bool {
	return errors.Is(err, target)
}

// Builder provides a fluent interface for constructing errors.
type Builder struct {
	code    Code
	op      string
	err     error
	msg     string
	details []any
}

// Build constructs a new error with the builder's configuration.
func Build(code Code) *Builder {
	return &Builder{code: code}
}

// WithOp sets the operation that failed.
func (b *Builder) WithOp(op string) *Builder {
	b.op = op
	return b
}

// WithErr wraps an underlying error.
func (b *Builder) WithErr(err error) *Builder {
	b.err = err
	return b
}

// WithMsg sets a human-readable message.
func (b *Builder) WithMsg(msg string) *Builder {
	b.msg = msg
	return b
}

// WithMsgf sets a formatted human-readable message.
func (b *Builder) WithMsgf(format string, args ...any) *Builder {
	b.msg = fmt.Sprintf(format, args...)
	return b
}

// WithDetails adds structured details to the error.
func (b *Builder) WithDetails(details ...any) *Builder {
	b.details = append(b.details, details...)
	return b
}

// Err builds and returns the error.
func (b *Builder) Err() error {
	return &E{
		Code:    b.code,
		Op:      b.op,
		Err:     b.err,
		Msg:     b.msg,
		Details: b.details,
	}
}
