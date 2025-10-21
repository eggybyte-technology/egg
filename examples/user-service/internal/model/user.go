// Package model provides data models for the user service.
//
// Overview:
//   - Responsibility: Define database models and data structures
//   - Key Types: User model with GORM annotations
//   - Concurrency Model: Thread-safe database operations
//   - Error Semantics: Database errors are wrapped and returned
//   - Performance Notes: Optimized for PostgreSQL with proper indexing
//
// Usage:
//
//	user := &User{Email: "user@example.com", Name: "John Doe"}
//	db.Create(user)
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents a user entity in the database.
// It includes GORM annotations for automatic table creation and indexing.
type User struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Email     string    `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Name      string    `gorm:"not null;size:255" json:"name"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName returns the table name for the User model.
// This ensures consistent table naming across environments.
func (User) TableName() string {
	return "users"
}

// BeforeCreate is called before creating a new user record.
// It generates a UUID if one is not provided.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// Validate performs basic validation on the user data.
// Returns an error if validation fails.
func (u *User) Validate() error {
	if u.Email == "" {
		return ErrInvalidEmail
	}
	if u.Name == "" {
		return ErrInvalidName
	}
	return nil
}
