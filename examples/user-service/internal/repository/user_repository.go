// Package repository provides data access layer for the user service.
//
// Overview:
//   - Responsibility: Database operations and data persistence
//   - Key Types: UserRepository interface and implementation
//   - Concurrency Model: Thread-safe database operations with context
//   - Error Semantics: Database errors are wrapped and returned
//   - Performance Notes: Optimized queries with proper indexing
//
// Usage:
//
//	repo := NewUserRepository(db)
//	user, err := repo.Create(ctx, &User{Email: "user@example.com"})
package repository

import (
	"context"

	"go.eggybyte.com/egg/core/errors"
	"go.eggybyte.com/egg/examples/user-service/internal/model"
	"gorm.io/gorm"
)

// UserRepository defines the interface for user data operations.
// All methods are context-aware and return structured errors.
type UserRepository interface {
	// Create creates a new user in the database.
	// Returns the created user with generated ID and timestamps.
	Create(ctx context.Context, user *model.User) (*model.User, error)

	// GetByID retrieves a user by their ID.
	// Returns ErrUserNotFound if user doesn't exist.
	GetByID(ctx context.Context, id string) (*model.User, error)

	// GetByEmail retrieves a user by their email address.
	// Returns ErrUserNotFound if user doesn't exist.
	GetByEmail(ctx context.Context, email string) (*model.User, error)

	// Update updates an existing user in the database.
	// Returns ErrUserNotFound if user doesn't exist.
	Update(ctx context.Context, user *model.User) (*model.User, error)

	// Delete removes a user from the database by ID.
	// Returns ErrUserNotFound if user doesn't exist.
	Delete(ctx context.Context, id string) error

	// List retrieves users with pagination.
	// Returns empty list if no users found.
	List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error)

	// DeleteAll deletes all users from the database.
	DeleteAll(ctx context.Context) (int64, error)
}

// userRepository implements the UserRepository interface using GORM.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository instance.
// The returned repository is safe for concurrent use.
//
// Parameters:
//   - db: GORM database instance (must not be nil)
//
// Returns:
//   - UserRepository: The created repository instance
//
// Panics:
//   - If db is nil (fail-fast at startup)
//
// Rationale:
// This function panics on nil database rather than returning an error
// because this is a startup-time issue that should never occur in production.
// If the database is nil, the repository cannot function and should not start.
func NewUserRepository(db *gorm.DB) UserRepository {
	if db == nil {
		panic("NewUserRepository: database cannot be nil")
	}

	return &userRepository{db: db}
}

// Create creates a new user in the database with validation and duplicate checking.
//
// This method performs the following operations:
//  1. Validates the user model using user.Validate()
//  2. Checks for duplicate email addresses
//  3. Generates UUID and timestamps via GORM hooks
//  4. Inserts the record into the database
//
// Parameters:
//   - ctx: request context for cancellation and deadlines
//   - user: user model to create (ID will be auto-generated if empty)
//
// Returns:
//   - *model.User: created user with generated ID and timestamps
//   - error: nil on success; wrapped error on failure
//   - CodeInvalidArgument: validation failed
//   - CodeAlreadyExists: email already registered
//   - CodeInternal: database operation failed
//
// Concurrency:
//
//	Safe for concurrent use. Database handles constraint violations atomically.
func (r *userRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	if err := user.Validate(); err != nil {
		return nil, errors.Wrap(errors.CodeInvalidArgument, "user validation", err)
	}

	// Check if email already exists
	var existingUser model.User
	if err := r.db.WithContext(ctx).Where("email = ?", user.Email).First(&existingUser).Error; err == nil {
		return nil, errors.Wrap(errors.CodeAlreadyExists, "email check", model.ErrEmailExists)
	} else if err != gorm.ErrRecordNotFound {
		return nil, errors.Wrap(errors.CodeInternal, "email check", err)
	}

	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "create user", err)
	}

	return user, nil
}

// GetByID retrieves a user by their unique ID.
//
// Parameters:
//   - ctx: request context for cancellation and deadlines
//   - id: user UUID to lookup
//
// Returns:
//   - *model.User: found user instance
//   - error: nil on success; wrapped error on failure
//   - CodeNotFound: user does not exist
//   - CodeInternal: database query failed
//
// Concurrency:
//
//	Safe for concurrent use. Read-only operation with context support.
func (r *userRepository) GetByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.Wrap(errors.CodeNotFound, "get user by id", model.ErrUserNotFound)
		}
		return nil, errors.Wrap(errors.CodeInternal, "get user by id", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by their email address.
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.Wrap(errors.CodeNotFound, "get user by email", model.ErrUserNotFound)
		}
		return nil, errors.Wrap(errors.CodeInternal, "get user by email", err)
	}

	return &user, nil
}

// Update updates an existing user in the database.
func (r *userRepository) Update(ctx context.Context, user *model.User) (*model.User, error) {
	if err := user.Validate(); err != nil {
		return nil, errors.Wrap(errors.CodeInvalidArgument, "user validation", err)
	}

	// Check if user exists
	var existingUser model.User
	if err := r.db.WithContext(ctx).First(&existingUser, "id = ?", user.ID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.Wrap(errors.CodeNotFound, "check user existence", model.ErrUserNotFound)
		}
		return nil, errors.Wrap(errors.CodeInternal, "check user existence", err)
	}

	// Check if email is being changed and if new email already exists
	if existingUser.Email != user.Email {
		var emailUser model.User
		if err := r.db.WithContext(ctx).Where("email = ? AND id != ?", user.Email, user.ID).First(&emailUser).Error; err == nil {
			return nil, errors.Wrap(errors.CodeAlreadyExists, "email check", model.ErrEmailExists)
		} else if err != gorm.ErrRecordNotFound {
			return nil, errors.Wrap(errors.CodeInternal, "email check", err)
		}
	}

	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return nil, errors.Wrap(errors.CodeInternal, "update user", err)
	}

	return user, nil
}

// Delete removes a user from the database by ID.
func (r *userRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.User{}, "id = ?", id)
	if result.Error != nil {
		return errors.Wrap(errors.CodeInternal, "delete user", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.Wrap(errors.CodeNotFound, "get user by id", model.ErrUserNotFound)
	}

	return nil
}

// List retrieves users with pagination and automatic parameter normalization.
//
// This method performs efficient paginated retrieval with:
//  1. Parameter validation and normalization
//  2. Total count query (for pagination metadata)
//  3. Paginated data query with ordering
//
// Parameters:
//   - ctx: request context for cancellation and deadlines
//   - page: page number (1-indexed); auto-corrected to 1 if < 1
//   - pageSize: items per page; auto-corrected to [1, 100] range
//
// Returns:
//   - []*model.User: slice of users for the requested page (may be empty)
//   - int64: total count of users in database
//   - error: nil on success; wrapped error with CodeInternal on database failure
//
// Behavior:
//   - Default page size: 10
//   - Maximum page size: 100 (prevents excessive memory usage)
//   - Ordering: created_at DESC (newest first)
//   - Empty result: returns empty slice (not nil) with total count
//
// Performance:
//   - Two queries: COUNT(*) + SELECT with LIMIT/OFFSET
//   - Index hint: Uses primary key and created_at index
//   - Memory: O(pageSize) per call
//
// Concurrency:
//
//	Safe for concurrent use. Read-only operations with context support.
func (r *userRepository) List(ctx context.Context, page, pageSize int) ([]*model.User, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	var users []*model.User
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternal, "count users", err)
	}

	// Get users with pagination
	if err := r.db.WithContext(ctx).Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, errors.Wrap(errors.CodeInternal, "list users", err)
	}

	return users, total, nil
}

// DeleteAll deletes all users from the database.
func (r *userRepository) DeleteAll(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).Where("1 = 1").Delete(&model.User{})
	if result.Error != nil {
		return 0, errors.Wrap(errors.CodeInternal, "delete all users", result.Error)
	}
	return result.RowsAffected, nil
}
