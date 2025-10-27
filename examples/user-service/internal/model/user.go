// Package model provides domain models and data structures for the user service.
//
// Overview:
//   - Responsibility: Define domain entities, validation rules, and GORM mappings
//   - Key Types: User model with GORM annotations and validation logic
//   - Concurrency Model: Models are safe for concurrent use; validation is stateless
//   - Error Semantics: Validation errors use predefined domain errors from errors.go
//   - Performance Notes: Optimized for MySQL/PostgreSQL with proper indexing and UUID primary keys
//
// Usage:
//
//	user := &model.User{
//	    Email: "user@example.com",
//	    Name:  "John Doe",
//	}
//	if err := user.Validate(); err != nil {
//	    return err
//	}
//	db.Create(user)
package model

import (
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user entity in the database.
//
// This model demonstrates production-ready entity design with:
//   - UUID primary key for distributed system compatibility
//   - Unique email constraint for business rule enforcement
//   - Automatic timestamp management via GORM hooks
//   - JSON serialization support for API responses
//
// GORM Configuration:
//   - Table: "users"
//   - Primary Key: UUID string (varchar(36))
//   - Unique Index: email
//   - Auto-managed: created_at, updated_at
//
// Concurrency:
//
//	Safe for concurrent use. Each instance represents immutable data after creation.
type User struct {
	// ID is the unique identifier for the user (UUID v4 format).
	// Generated automatically before create if not provided.
	ID string `gorm:"primaryKey;type:varchar(36)" json:"id"`

	// Email is the user's email address (unique constraint).
	// Must be valid email format and unique across all users.
	Email string `gorm:"uniqueIndex;not null;size:255" json:"email"`

	// Name is the user's display name.
	// Required field, maximum 255 characters.
	Name string `gorm:"not null;size:255" json:"name"`

	// CreatedAt is the timestamp when the user was created.
	// Managed automatically by GORM.
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// UpdatedAt is the timestamp when the user was last updated.
	// Managed automatically by GORM on any update operation.
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// emailRegex defines a basic email validation pattern following RFC 5322 simplified rules.
//
// This regex validates that the email has:
//   - Local part: alphanumeric plus . _ % + -
//   - @ symbol
//   - Domain part: alphanumeric plus . -
//   - TLD: at least 2 characters
//
// Note: This is a simplified pattern for common use cases. For strict RFC 5322
// compliance, consider using a dedicated email validation library.
//
// Concurrency:
//
//	Safe for concurrent use (compiled once, read-only access).
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// TableName returns the table name for the User model.
//
// This method implements the GORM Tabler interface to specify a custom table name.
// Using explicit table names ensures consistency across different environments and
// avoids GORM's default pluralization behavior.
//
// Returns:
//   - string: The table name "users"
//
// Concurrency:
//
//	Safe for concurrent use (stateless method).
func (User) TableName() string {
	return "users"
}

// BeforeCreate is a GORM hook called before creating a new user record.
//
// This hook automatically generates a UUID v4 for the ID field if not already set.
// This ensures every user has a unique, distributed-system-friendly identifier.
//
// Parameters:
//   - tx: GORM database transaction (not used but required by interface)
//
// Returns:
//   - error: Always returns nil (UUID generation cannot fail)
//
// Concurrency:
//
//	Called within database transaction; safe for concurrent use.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// Validate performs comprehensive validation on the user data before persistence.
//
// This method should be called before any database operation to ensure data integrity.
// It validates both required fields and format constraints.
//
// Validation rules:
//   - Email must not be empty
//   - Email must match a valid email format (RFC 5322 simplified pattern)
//   - Name must not be empty
//   - Name length is implicitly validated by GORM (max 255 chars)
//
// Returns:
//   - error: nil if validation passes; predefined domain error otherwise
//   - ErrInvalidEmail: if email is empty or has invalid format
//   - ErrInvalidName: if name is empty
//
// Examples:
//
//	user := &User{Email: "test@example.com", Name: "John"}
//	if err := user.Validate(); err != nil {
//	    // Handle validation error
//	}
//
// Concurrency:
//
//	Safe for concurrent use. Validation is stateless and uses read-only regex.
//
// Performance:
//
//	O(n) where n is the length of the email string. Regex is pre-compiled.
func (u *User) Validate() error {
	if u.Email == "" {
		return ErrInvalidEmail
	}

	// Validate email format using pre-compiled regex
	if !emailRegex.MatchString(u.Email) {
		return ErrInvalidEmail
	}

	if u.Name == "" {
		return ErrInvalidName
	}

	return nil
}
