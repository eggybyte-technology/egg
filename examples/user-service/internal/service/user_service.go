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
	"fmt"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/core/errors"
	"go.eggybyte.com/egg/core/log"
	greetv1 "go.eggybyte.com/egg/examples/minimal-connect-service/gen/go/greet/v1"
	userv1 "go.eggybyte.com/egg/examples/user-service/gen/go/user/v1"
	"go.eggybyte.com/egg/examples/user-service/internal/client"
	"go.eggybyte.com/egg/examples/user-service/internal/model"
	"go.eggybyte.com/egg/examples/user-service/internal/repository"
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

	// AdminResetAllUsers deletes all users (admin operation, requires internal token).
	AdminResetAllUsers(ctx context.Context, req *userv1.AdminResetAllUsersRequest) (*userv1.AdminResetAllUsersResponse, error)

	// GetGreeting demonstrates service-to-service communication using clientx.
	// This method calls the greet service and returns a personalized greeting.
	GetGreeting(ctx context.Context, userName string) (string, error)

	// ValidateInternalToken validates an internal token by calling the greet service.
	// This demonstrates service-to-service communication and token validation.
	ValidateInternalToken(ctx context.Context, req *userv1.ValidateInternalTokenRequest) (*userv1.ValidateInternalTokenResponse, error)
}

// userService implements the UserService interface.
type userService struct {
	repo        repository.UserRepository
	logger      log.Logger
	greetClient *client.GreetClient
}

// NewUserService creates a new UserService instance with dependency injection.
//
// This constructor demonstrates the egg framework's dependency injection pattern:
// accepting interfaces (not concrete types) to enable testability and flexibility.
//
// Parameters:
//   - repo: UserRepository implementation for data access (must not be nil)
//   - logger: Logger implementation for structured logging (must not be nil)
//   - greetClient: GreetClient for service-to-service communication (may be nil)
//
// Returns:
//   - UserService: service instance ready for use
//
// Panics:
//   - If repo is nil (fail-fast at startup)
//   - If logger is nil (fail-fast at startup)
//
// Rationale:
//
// This function panics on nil dependencies rather than returning an error because
// these are startup-time configuration issues that should never occur in production.
// Panicking during initialization is preferable to runtime nil pointer dereferences.
//
// Concurrency:
//
//	Returned service is safe for concurrent use across multiple goroutines.
func NewUserService(repo repository.UserRepository, logger log.Logger, greetClient *client.GreetClient) UserService {
	if repo == nil {
		panic("NewUserService: repository cannot be nil")
	}
	if logger == nil {
		panic("NewUserService: logger cannot be nil")
	}

	return &userService{
		repo:        repo,
		logger:      logger,
		greetClient: greetClient,
	}
}

// CreateUser creates a new user with comprehensive business validation.
//
// This method implements multi-layer validation:
//  1. Request field validation (required fields)
//  2. Domain model validation (format, constraints)
//  3. Business rule validation (uniqueness via repository)
//
// Parameters:
//   - ctx: request context for cancellation and deadlines
//   - req: protobuf request containing email and name
//
// Returns:
//   - *userv1.CreateUserResponse: response with created user details
//   - error: nil on success; structured error on failure
//   - CodeInvalidArgument: validation failed (empty/invalid fields)
//   - CodeAlreadyExists: email already registered
//   - CodeInternal: database operation failed
//
// Behavior:
//   - Generates UUID for new user automatically
//   - Sets created_at and updated_at timestamps
//   - Logs validation and operation steps at DEBUG level
//
// Concurrency:
//
//	Safe for concurrent use. Database handles race conditions with unique constraints.
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

// AdminResetAllUsers deletes all users from the database.
// This is a destructive admin operation that requires internal token authentication.
//
// Parameters:
//   - ctx: request context (must contain valid internal token)
//   - req: request with confirmation flag
//
// Returns:
//   - *userv1.AdminResetAllUsersResponse: count of deleted users
//   - error: nil on success; CodeInvalidArgument if not confirmed
//
// Security:
//   - This method MUST be protected by internal token validation at handler level
//   - Should only be callable by internal services
func (s *userService) AdminResetAllUsers(ctx context.Context, req *userv1.AdminResetAllUsersRequest) (*userv1.AdminResetAllUsersResponse, error) {
	s.logger.Debug("AdminResetAllUsers started", log.Bool("confirm", req.Confirm))

	// Require explicit confirmation
	if !req.Confirm {
		s.logger.Debug("AdminResetAllUsers rejected: not confirmed")
		return nil, errors.New(errors.CodeInvalidArgument, "confirm must be true to execute")
	}

	s.logger.Debug("AdminResetAllUsers calling repository")

	// Delete all users
	count, err := s.repo.DeleteAll(ctx)
	if err != nil {
		s.logger.Debug("AdminResetAllUsers repository failed", log.Str("error", err.Error()))
		return nil, err
	}

	s.logger.Info("AdminResetAllUsers completed",
		log.Int("deleted_count", int(count)))

	return &userv1.AdminResetAllUsersResponse{
		DeletedCount: int32(count),
		Success:      true,
	}, nil
}

// GetGreeting demonstrates service-to-service communication using clientx.
//
// This method showcases the egg framework's recommended pattern for calling other
// microservices:
//   - Uses clientx.NewConnectClient for production-ready client features
//   - Automatic retry with exponential backoff
//   - Circuit breaker to prevent cascade failures
//   - Internal token injection for service-to-service authentication
//   - Proper error handling and logging
//
// Parameters:
//   - ctx: request context for cancellation and deadlines
//   - userName: name to greet
//
// Returns:
//   - string: greeting message from the greet service
//   - error: nil on success; wrapped error on failure
//
// Errors:
//   - CodeUnavailable: greet service is unavailable (circuit breaker or network error)
//   - CodeDeadlineExceeded: request timeout exceeded
//
// Concurrency:
//   - Safe for concurrent use
func (s *userService) GetGreeting(ctx context.Context, userName string) (string, error) {
	s.logger.Debug("GetGreeting started", log.Str("user_name", userName))

	// Check if greet client is configured
	if s.greetClient == nil {
		s.logger.Debug("GetGreeting skipped: greet client not configured")
		return "", errors.New(errors.CodeUnimplemented, "greet service not configured")
	}

	// Call the greet service using clientx client
	// The client automatically handles:
	// - Retry with exponential backoff
	// - Circuit breaker
	// - Internal token injection
	// - Timeout handling
	req := connect.NewRequest(&greetv1.SayHelloRequest{
		Name:     userName,
		Language: "en",
	})

	resp, err := s.greetClient.Client().SayHello(ctx, req)
	if err != nil {
		s.logger.Debug("GetGreeting failed", log.Str("user_name", userName), log.Str("error", err.Error()))
		return "", errors.Wrap(errors.CodeUnavailable, "call greet service", err)
	}

	s.logger.Debug("GetGreeting completed",
		log.Str("user_name", userName),
		log.Str("greeting", resp.Msg.Message))

	return resp.Msg.Message, nil
}

// ValidateInternalToken validates an internal token by calling the greet service.
//
// This method demonstrates standard client usage pattern - using the injected greetClient
// to validate that the service's internal token works by making a real service call.
// This is the recommended pattern: always use injected clients, never create temporary clients.
//
// Parameters:
//   - ctx: request context (must not be nil)
//   - req: validation request containing the token to validate
//
// Returns:
//   - *userv1.ValidateInternalTokenResponse: validation result with status and message
//   - error: nil on success; wrapped error on failure
//
// Behavior:
//   - Uses the standard injected greetClient (configured with service's internal token)
//   - Makes a real service call to validate token functionality
//   - Returns validation result based on whether the call succeeds
//
// Concurrency:
//   - Safe for concurrent use
func (s *userService) ValidateInternalToken(ctx context.Context, req *userv1.ValidateInternalTokenRequest) (*userv1.ValidateInternalTokenResponse, error) {
	s.logger.Debug("ValidateInternalToken started")

	// Check if greet client is configured
	if s.greetClient == nil {
		s.logger.Debug("ValidateInternalToken: greet client not configured")
		return &userv1.ValidateInternalTokenResponse{
			Valid:        false,
			ErrorMessage: "greet service not configured",
		}, nil
	}

	// Use the standard injected client to call greet service
	// The client already has the service's internal token configured
	// This demonstrates standard client usage pattern
	greetReq := connect.NewRequest(&greetv1.SayHelloRequest{
		Name:     "Token Validator",
		Language: "en",
	})

	resp, err := s.greetClient.Client().SayHello(ctx, greetReq)
	if err != nil {
		s.logger.Debug("ValidateInternalToken: greet service call failed",
			log.Str("error", err.Error()))
		return &userv1.ValidateInternalTokenResponse{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("validation failed: %v", err),
		}, nil
	}

	s.logger.Debug("ValidateInternalToken: validation successful",
		log.Str("message", resp.Msg.Message))

	return &userv1.ValidateInternalTokenResponse{
		Valid:   true,
		Message: resp.Msg.Message,
	}, nil
}

// toProtoUser converts a domain User model to protobuf User message.
//
// This helper function eliminates code duplication across service methods and
// centralizes the mapping logic between domain and API layers.
//
// Parameters:
//   - user: domain user model (must not be nil)
//
// Returns:
//   - *userv1.User: protobuf user message with Unix timestamps
//
// Behavior:
//   - Timestamps are converted from time.Time to Unix epoch seconds
//   - All string fields are copied directly
//   - No validation is performed (assumes valid domain model)
//
// Concurrency:
//
//	Safe for concurrent use (pure function, no shared state).
func toProtoUser(user *model.User) *userv1.User {
	return &userv1.User{
		Id:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.Unix(),
		UpdatedAt: user.UpdatedAt.Unix(),
	}
}

// normalizePage ensures page number is at least 1 for safe pagination.
//
// Parameters:
//   - page: requested page number (may be negative or zero)
//
// Returns:
//   - int: normalized page number (minimum 1)
//
// Concurrency:
//
//	Safe for concurrent use (pure function).
func normalizePage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

// normalizePageSize ensures page size is within valid range [1, 100].
//
// This function prevents:
//   - Excessive memory usage from large page sizes
//   - Invalid queries from zero or negative page sizes
//
// Parameters:
//   - pageSize: requested page size (may be any integer)
//
// Returns:
//   - int: normalized page size
//   - Returns 10 if pageSize < 1 (default)
//   - Returns 100 if pageSize > 100 (max)
//   - Returns pageSize otherwise
//
// Concurrency:
//
//	Safe for concurrent use (pure function).
func normalizePageSize(pageSize int) int {
	if pageSize < 1 {
		return 10 // default
	}
	if pageSize > 100 {
		return 100 // max
	}
	return pageSize
}
