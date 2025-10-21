// Package service provides business logic for the user service.
//
// Overview:
//   - Responsibility: Business logic and domain operations
//   - Key Types: UserService interface and implementation
//   - Concurrency Model: Thread-safe service operations
//   - Error Semantics: Domain errors are wrapped and returned
//   - Performance Notes: Optimized for high-throughput operations
//
// Usage:
//
//	service := NewUserService(repo)
//	user, err := service.CreateUser(ctx, &CreateUserRequest{Email: "user@example.com"})
package service

import (
	"context"

	"github.com/eggybyte-technology/egg/core/errors"
	"github.com/eggybyte-technology/egg/core/log"
	userv1 "github.com/eggybyte-technology/egg/examples/user-service/gen/go/user/v1"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/model"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/repository"
)

// UserService defines the interface for user business operations.
// All methods are context-aware and return structured errors.
type UserService interface {
	// CreateUser creates a new user with business validation.
	CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error)

	// GetUser retrieves a user by ID with proper error handling.
	GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error)

	// UpdateUser updates an existing user with validation.
	UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error)

	// DeleteUser removes a user by ID.
	DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error)

	// ListUsers retrieves users with pagination.
	ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error)
}

// userService implements the UserService interface.
type userService struct {
	repo   repository.UserRepository
	logger log.Logger
}

// NewUserService creates a new UserService instance.
// The returned service is safe for concurrent use.
func NewUserService(repo repository.UserRepository, logger log.Logger) UserService {
	return &userService{
		repo:   repo,
		logger: logger,
	}
}

// CreateUser creates a new user with business validation.
func (s *userService) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	// Check if repository is available
	if s.repo == nil {
		return nil, errors.New("SERVICE_UNAVAILABLE", "database repository not available")
	}

	// Validate request
	if req.Email == "" {
		return nil, errors.New("INVALID_REQUEST", "email is required")
	}
	if req.Name == "" {
		return nil, errors.New("INVALID_REQUEST", "name is required")
	}

	// Create user model
	user := &model.User{
		Email: req.Email,
		Name:  req.Name,
	}

	// Create user in repository
	createdUser, err := s.repo.Create(ctx, user)
	if err != nil {
		s.logger.Error(err, "Failed to create user", log.Str("email", req.Email))
		return nil, err
	}

	s.logger.Info("User created successfully",
		log.Str("user_id", createdUser.ID),
		log.Str("email", createdUser.Email))

	// Convert to response
	response := &userv1.CreateUserResponse{
		User: &userv1.User{
			Id:        createdUser.ID,
			Email:     createdUser.Email,
			Name:      createdUser.Name,
			CreatedAt: createdUser.CreatedAt.Unix(),
			UpdatedAt: createdUser.UpdatedAt.Unix(),
		},
	}

	return response, nil
}

// GetUser retrieves a user by ID with proper error handling.
func (s *userService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	// Check if repository is available
	if s.repo == nil {
		return nil, errors.New("SERVICE_UNAVAILABLE", "database repository not available")
	}

	// Validate request
	if req.Id == "" {
		return nil, errors.New("INVALID_REQUEST", "user ID is required")
	}

	// Get user from repository
	user, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		s.logger.Error(err, "Failed to get user", log.Str("user_id", req.Id))
		return nil, err
	}

	s.logger.Info("User retrieved successfully", log.Str("user_id", user.ID))

	// Convert to response
	response := &userv1.GetUserResponse{
		User: &userv1.User{
			Id:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		},
	}

	return response, nil
}

// UpdateUser updates an existing user with validation.
func (s *userService) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	// Check if repository is available
	if s.repo == nil {
		return nil, errors.New("SERVICE_UNAVAILABLE", "database repository not available")
	}

	// Validate request
	if req.Id == "" {
		return nil, errors.New("INVALID_REQUEST", "user ID is required")
	}
	if req.Email == "" {
		return nil, errors.New("INVALID_REQUEST", "email is required")
	}
	if req.Name == "" {
		return nil, errors.New("INVALID_REQUEST", "name is required")
	}

	// Create user model
	user := &model.User{
		ID:    req.Id,
		Email: req.Email,
		Name:  req.Name,
	}

	// Update user in repository
	updatedUser, err := s.repo.Update(ctx, user)
	if err != nil {
		s.logger.Error(err, "Failed to update user", log.Str("user_id", req.Id))
		return nil, err
	}

	s.logger.Info("User updated successfully",
		log.Str("user_id", updatedUser.ID),
		log.Str("email", updatedUser.Email))

	// Convert to response
	response := &userv1.UpdateUserResponse{
		User: &userv1.User{
			Id:        updatedUser.ID,
			Email:     updatedUser.Email,
			Name:      updatedUser.Name,
			CreatedAt: updatedUser.CreatedAt.Unix(),
			UpdatedAt: updatedUser.UpdatedAt.Unix(),
		},
	}

	return response, nil
}

// DeleteUser removes a user by ID.
func (s *userService) DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error) {
	// Check if repository is available
	if s.repo == nil {
		return nil, errors.New("SERVICE_UNAVAILABLE", "database repository not available")
	}

	// Validate request
	if req.Id == "" {
		return nil, errors.New("INVALID_REQUEST", "user ID is required")
	}

	// Delete user from repository
	err := s.repo.Delete(ctx, req.Id)
	if err != nil {
		s.logger.Error(err, "Failed to delete user", log.Str("user_id", req.Id))
		return nil, err
	}

	s.logger.Info("User deleted successfully", log.Str("user_id", req.Id))

	// Convert to response
	response := &userv1.DeleteUserResponse{
		Success: true,
	}

	return response, nil
}

// ListUsers retrieves users with pagination.
func (s *userService) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	// Check if repository is available
	if s.repo == nil {
		return nil, errors.New("SERVICE_UNAVAILABLE", "database repository not available")
	}

	// Set default pagination
	page := int(req.Page)
	if page < 1 {
		page = 1
	}

	pageSize := int(req.PageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// Get users from repository
	users, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error(err, "Failed to list users",
			log.Int("page", page),
			log.Int("page_size", pageSize))
		return nil, err
	}

	s.logger.Info("Users listed successfully",
		log.Int("count", len(users)),
		log.Int("total", int(total)),
		log.Int("page", page))

	// Convert to response
	responseUsers := make([]*userv1.User, len(users))
	for i, user := range users {
		responseUsers[i] = &userv1.User{
			Id:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			CreatedAt: user.CreatedAt.Unix(),
			UpdatedAt: user.UpdatedAt.Unix(),
		}
	}

	response := &userv1.ListUsersResponse{
		Users:    responseUsers,
		Total:    int32(total),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}

	return response, nil
}
