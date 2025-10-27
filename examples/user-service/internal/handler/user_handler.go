// Package handler provides Connect protocol handlers for the user service.
//
// Overview:
//   - Responsibility: Connect protocol implementation and request handling
//   - Key Types: UserHandler struct with Connect method implementations
//   - Concurrency Model: Thread-safe handlers with context propagation
//   - Error Semantics: Connect errors are properly mapped and returned
//   - Performance Notes: Optimized for high-throughput Connect requests
//
// Usage:
//
//	handler := NewUserHandler(service)
//	connectHandler := userv1connect.NewUserServiceHandler(handler)
package handler

import (
	"context"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/core/log"
	userv1 "go.eggybyte.com/egg/examples/user-service/gen/go/user/v1"
	userv1connect "go.eggybyte.com/egg/examples/user-service/gen/go/user/v1/userv1connect"
	"go.eggybyte.com/egg/examples/user-service/internal/service"
)

// UserHandler implements the Connect protocol for the user service.
// It bridges the Connect protocol with the business logic service.
type UserHandler struct {
	userv1connect.UnimplementedUserServiceHandler

	service service.UserService
	logger  log.Logger
}

// NewUserHandler creates a new UserHandler instance.
// The returned handler is safe for concurrent use.
//
// Parameters:
//   - service: UserService implementation (must not be nil)
//   - logger: Logger instance (must not be nil)
//
// Returns:
//   - *UserHandler: The created handler instance
//
// Panics:
//   - If service is nil (fail-fast at startup)
//   - If logger is nil (fail-fast at startup)
//
// Rationale:
// This function panics on nil dependencies rather than returning an error
// because these are startup-time issues that should never occur in production.
// If dependencies are nil, the handler cannot function and should not start.
func NewUserHandler(service service.UserService, logger log.Logger) *UserHandler {
	if service == nil {
		panic("NewUserHandler: service cannot be nil")
	}
	if logger == nil {
		panic("NewUserHandler: logger cannot be nil")
	}

	return &UserHandler{
		service: service,
		logger:  logger,
	}
}

// CreateUser handles CreateUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) CreateUser(ctx context.Context, req *connect.Request[userv1.CreateUserRequest]) (*connect.Response[userv1.CreateUserResponse], error) {
	// Only log at DEBUG level for intermediate steps - Connect interceptor handles request lifecycle
	h.logger.Debug("CreateUser processing",
		log.Str("email", req.Msg.Email),
		log.Str("name", req.Msg.Name))

	// Delegate to business service
	response, err := h.service.CreateUser(ctx, req.Msg)
	if err != nil {
		// Error logging is handled by Connect interceptor with proper error classification
		return nil, err
	}

	// Success logging is handled by Connect interceptor
	h.logger.Debug("CreateUser completed",
		log.Str("user_id", response.User.Id))

	return connect.NewResponse(response), nil
}

// GetUser handles GetUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) GetUser(ctx context.Context, req *connect.Request[userv1.GetUserRequest]) (*connect.Response[userv1.GetUserResponse], error) {
	// Only log at DEBUG level for intermediate steps - Connect interceptor handles request lifecycle
	h.logger.Debug("GetUser processing",
		log.Str("user_id", req.Msg.Id))

	// Delegate to business service
	response, err := h.service.GetUser(ctx, req.Msg)
	if err != nil {
		// Error logging is handled by Connect interceptor with proper error classification
		return nil, err
	}

	// Success logging is handled by Connect interceptor
	h.logger.Debug("GetUser completed",
		log.Str("user_id", response.User.Id))

	return connect.NewResponse(response), nil
}

// UpdateUser handles UpdateUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) UpdateUser(ctx context.Context, req *connect.Request[userv1.UpdateUserRequest]) (*connect.Response[userv1.UpdateUserResponse], error) {
	// Only log at DEBUG level for intermediate steps - Connect interceptor handles request lifecycle
	h.logger.Debug("UpdateUser processing",
		log.Str("user_id", req.Msg.Id),
		log.Str("email", req.Msg.Email))

	// Delegate to business service
	response, err := h.service.UpdateUser(ctx, req.Msg)
	if err != nil {
		// Error logging is handled by Connect interceptor with proper error classification
		return nil, err
	}

	// Success logging is handled by Connect interceptor
	h.logger.Debug("UpdateUser completed",
		log.Str("user_id", response.User.Id))

	return connect.NewResponse(response), nil
}

// DeleteUser handles DeleteUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) DeleteUser(ctx context.Context, req *connect.Request[userv1.DeleteUserRequest]) (*connect.Response[userv1.DeleteUserResponse], error) {
	// Only log at DEBUG level for intermediate steps - Connect interceptor handles request lifecycle
	h.logger.Debug("DeleteUser processing",
		log.Str("user_id", req.Msg.Id))

	// Delegate to business service
	response, err := h.service.DeleteUser(ctx, req.Msg)
	if err != nil {
		// Error logging is handled by Connect interceptor with proper error classification
		return nil, err
	}

	// Success logging is handled by Connect interceptor
	h.logger.Debug("DeleteUser completed",
		log.Str("user_id", req.Msg.Id))

	return connect.NewResponse(response), nil
}

// ListUsers handles ListUsers Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) ListUsers(ctx context.Context, req *connect.Request[userv1.ListUsersRequest]) (*connect.Response[userv1.ListUsersResponse], error) {
	// Only log at DEBUG level for intermediate steps - Connect interceptor handles request lifecycle
	h.logger.Debug("ListUsers processing",
		log.Int("page", int(req.Msg.Page)),
		log.Int("page_size", int(req.Msg.PageSize)))

	// Delegate to business service
	response, err := h.service.ListUsers(ctx, req.Msg)
	if err != nil {
		// Error logging is handled by Connect interceptor with proper error classification
		return nil, err
	}

	// Success logging is handled by Connect interceptor
	h.logger.Debug("ListUsers completed",
		log.Int("count", len(response.Users)),
		log.Int("total", int(response.Total)))

	return connect.NewResponse(response), nil
}
