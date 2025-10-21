// Package model provides error definitions for the user service.
//
// Overview:
//   - Responsibility: Define domain-specific error types
//   - Key Types: Error constants and custom error types
//   - Concurrency Model: Safe for concurrent use
//   - Error Semantics: Structured error handling with codes
//   - Performance Notes: Lightweight error definitions
//
// Usage:
//
//	return ErrUserNotFound
//	return ErrInvalidEmail
package model

import "github.com/eggybyte-technology/egg/core/errors"

// Domain error codes for the user service.
const (
	ErrCodeUserNotFound  = "USER_NOT_FOUND"
	ErrCodeInvalidEmail  = "INVALID_EMAIL"
	ErrCodeInvalidName   = "INVALID_NAME"
	ErrCodeEmailExists   = "EMAIL_EXISTS"
	ErrCodeDatabaseError = "DATABASE_ERROR"
)

// Predefined errors for common scenarios.
var (
	ErrUserNotFound  = errors.New(ErrCodeUserNotFound, "user not found")
	ErrInvalidEmail  = errors.New(ErrCodeInvalidEmail, "invalid email address")
	ErrInvalidName   = errors.New(ErrCodeInvalidName, "invalid name")
	ErrEmailExists   = errors.New(ErrCodeEmailExists, "email already exists")
	ErrDatabaseError = errors.New(ErrCodeDatabaseError, "database operation failed")
)
