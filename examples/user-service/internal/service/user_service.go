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
//	service := NewUserService(repo, logger)
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
//
// Panics if repo or logger is nil (fail-fast at startup).
func NewUserService(repo repository.UserRepository, logger log.Logger) UserService {
	if repo == nil {
		panic("NewUserService: repository cannot be nil")
	}
	if logger == nil {
		panic("NewUserService: logger cannot be nil")
	}

	return &userService{
		repo:   repo,
		logger: logger,
	}
}

// CreateUser creates a new user with business validation.
func (s *userService) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	s.logger.Debug("CreateUser started", log.Str("email", req.Email), log.Str("name", req.Name))

	// Validate request fields
	if req.Email == "" {
		s.logger.Debug("CreateUser validation failed: empty email")
		return nil, errors.New(errors.CodeInvalidArgument, "email is required")
	}
	if req.Name == "" {
		s.logger.Debug("CreateUser validation failed: empty name")
		return nil, errors.New(errors.CodeInvalidArgument, "name is required")
	}

	// Create and validate user model
	user := &model.User{
		Email: req.Email,
		Name:  req.Name,
	}

	if err := user.Validate(); err != nil {
		s.logger.Debug("CreateUser model validation failed", log.Str("email", req.Email), log.Str("error", err.Error()))
		return nil, err
	}

	s.logger.Debug("CreateUser calling repository", log.Str("email", req.Email))

	// Create user in repository
	createdUser, err := s.repo.Create(ctx, user)
	if err != nil {
		s.logger.Debug("CreateUser repository failed", log.Str("email", req.Email), log.Str("error", err.Error()))
		return nil, err
	}

	s.logger.Debug("CreateUser completed successfully",
		log.Str("user_id", createdUser.ID),
		log.Str("email", createdUser.Email))

	return &userv1.CreateUserResponse{
		User: toProtoUser(createdUser),
	}, nil
}

// GetUser retrieves a user by ID with proper error handling.
func (s *userService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	s.logger.Debug("GetUser started", log.Str("user_id", req.Id))

	if req.Id == "" {
		s.logger.Debug("GetUser validation failed: empty ID")
		return nil, errors.New(errors.CodeInvalidArgument, "user ID is required")
	}

	s.logger.Debug("GetUser calling repository", log.Str("user_id", req.Id))

	user, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		s.logger.Debug("GetUser repository failed", log.Str("user_id", req.Id), log.Str("error", err.Error()))
		return nil, err
	}

	s.logger.Debug("GetUser completed successfully",
		log.Str("user_id", user.ID),
		log.Str("email", user.Email))

	return &userv1.GetUserResponse{
		User: toProtoUser(user),
	}, nil
}

// UpdateUser updates an existing user with validation.
func (s *userService) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
	s.logger.Debug("UpdateUser started",
		log.Str("user_id", req.Id),
		log.Str("email", req.Email),
		log.Str("name", req.Name))

	// Validate request fields
	if req.Id == "" {
		s.logger.Debug("UpdateUser validation failed: empty ID")
		return nil, errors.New(errors.CodeInvalidArgument, "user ID is required")
	}
	if req.Email == "" {
		s.logger.Debug("UpdateUser validation failed: empty email")
		return nil, errors.New(errors.CodeInvalidArgument, "email is required")
	}
	if req.Name == "" {
		s.logger.Debug("UpdateUser validation failed: empty name")
		return nil, errors.New(errors.CodeInvalidArgument, "name is required")
	}

	s.logger.Debug("UpdateUser fetching existing user", log.Str("user_id", req.Id))

	// Get existing user to preserve timestamps
	existingUser, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		s.logger.Debug("UpdateUser get existing user failed", log.Str("user_id", req.Id), log.Str("error", err.Error()))
		return nil, err
	}

	// Build updated user model
	user := &model.User{
		ID:        req.Id,
		Email:     req.Email,
		Name:      req.Name,
		CreatedAt: existingUser.CreatedAt,
	}

	if err := user.Validate(); err != nil {
		s.logger.Debug("UpdateUser model validation failed", log.Str("user_id", req.Id), log.Str("error", err.Error()))
		return nil, err
	}

	s.logger.Debug("UpdateUser calling repository", log.Str("user_id", req.Id))

	updatedUser, err := s.repo.Update(ctx, user)
	if err != nil {
		s.logger.Debug("UpdateUser repository failed", log.Str("user_id", req.Id), log.Str("error", err.Error()))
		return nil, err
	}

	s.logger.Debug("UpdateUser completed successfully",
		log.Str("user_id", updatedUser.ID),
		log.Str("email", updatedUser.Email))

	return &userv1.UpdateUserResponse{
		User: toProtoUser(updatedUser),
	}, nil
}

// DeleteUser removes a user by ID.
func (s *userService) DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error) {
	s.logger.Debug("DeleteUser started", log.Str("user_id", req.Id))

	if req.Id == "" {
		s.logger.Debug("DeleteUser validation failed: empty ID")
		return nil, errors.New(errors.CodeInvalidArgument, "user ID is required")
	}

	s.logger.Debug("DeleteUser calling repository", log.Str("user_id", req.Id))

	if err := s.repo.Delete(ctx, req.Id); err != nil {
		s.logger.Debug("DeleteUser repository failed", log.Str("user_id", req.Id), log.Str("error", err.Error()))
		return nil, err
	}

	s.logger.Debug("DeleteUser completed successfully", log.Str("user_id", req.Id))

	return &userv1.DeleteUserResponse{
		Success: true,
	}, nil
}

// ListUsers retrieves users with pagination.
func (s *userService) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	// Normalize pagination parameters
	page := normalizePage(int(req.Page))
	pageSize := normalizePageSize(int(req.PageSize))

	s.logger.Debug("ListUsers started",
		log.Int("requested_page", int(req.Page)),
		log.Int("requested_page_size", int(req.PageSize)),
		log.Int("normalized_page", page),
		log.Int("normalized_page_size", pageSize))

	s.logger.Debug("ListUsers calling repository",
		log.Int("page", page),
		log.Int("page_size", pageSize))

	users, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Debug("ListUsers repository failed",
			log.Int("page", page),
			log.Int("page_size", pageSize),
			log.Str("error", err.Error()))
		return nil, err
	}

	s.logger.Debug("ListUsers completed successfully",
		log.Int("returned_count", len(users)),
		log.Int("total", int(total)),
		log.Int("page", page),
		log.Int("page_size", pageSize))

	// Convert users to proto
	protoUsers := make([]*userv1.User, len(users))
	for i, user := range users {
		protoUsers[i] = toProtoUser(user)
	}

	return &userv1.ListUsersResponse{
		Users:    protoUsers,
		Total:    int32(total),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}

// toProtoUser converts a domain User model to protobuf User message.
// This helper function eliminates code duplication across service methods.
func toProtoUser(user *model.User) *userv1.User {
	return &userv1.User{
		Id:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(),
	}
}

// normalizePage ensures page number is at least 1.
func normalizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

// normalizePageSize ensures page size is within valid range [1, 100].
func normalizePageSize(pageSize int) int {
	if pageSize < 1 {
		return 10 // default
	}
	if pageSize > 100 {
		return 100 // max
	}
	return pageSize
}
