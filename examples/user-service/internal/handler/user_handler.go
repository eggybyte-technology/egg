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
	"go.eggybyte.com/egg/servicex"
)

// UserHandler implements the Connect protocol for the user service.
// It bridges the Connect protocol with the business logic service.
type UserHandler struct {
	userv1connect.UnimplementedUserServiceHandler

	service       service.UserService
	logger        log.Logger
	internalToken string
}

// NewUserHandler creates a new UserHandler instance.
// The returned handler is safe for concurrent use.
//
// Parameters:
//   - service: UserService implementation (must not be nil)
//   - logger: Logger instance (must not be nil)
//   - internalToken: Internal token for authentication (may be empty)
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
func NewUserHandler(service service.UserService, logger log.Logger, internalToken string) *UserHandler {
	if service == nil {
		panic("NewUserHandler: service cannot be nil")
	}
	if logger == nil {
		panic("NewUserHandler: logger cannot be nil")
	}

	return &UserHandler{
		service:       service,
		logger:        logger,
		internalToken: internalToken,
	}
}

// CreateUser handles CreateUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) CreateUser(ctx context.Context, req *connect.Request[userv1.CreateUserRequest]) (*connect.Response[userv1.CreateUserResponse], error) {
	return servicex.CallService(h.service.CreateUser, h.logger, "CreateUser")(ctx, req)
}

// GetUser handles GetUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) GetUser(ctx context.Context, req *connect.Request[userv1.GetUserRequest]) (*connect.Response[userv1.GetUserResponse], error) {
	return servicex.CallService(h.service.GetUser, h.logger, "GetUser")(ctx, req)
}

// UpdateUser handles UpdateUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) UpdateUser(ctx context.Context, req *connect.Request[userv1.UpdateUserRequest]) (*connect.Response[userv1.UpdateUserResponse], error) {
	return servicex.CallService(h.service.UpdateUser, h.logger, "UpdateUser")(ctx, req)
}

// DeleteUser handles DeleteUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) DeleteUser(ctx context.Context, req *connect.Request[userv1.DeleteUserRequest]) (*connect.Response[userv1.DeleteUserResponse], error) {
	return servicex.CallService(h.service.DeleteUser, h.logger, "DeleteUser")(ctx, req)
}

// ListUsers handles ListUsers Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) ListUsers(ctx context.Context, req *connect.Request[userv1.ListUsersRequest]) (*connect.Response[userv1.ListUsersResponse], error) {
	return servicex.CallService(h.service.ListUsers, h.logger, "ListUsers")(ctx, req)
}

// AdminResetAllUsers handles AdminResetAllUsers Connect requests.
// This is a protected admin operation that REQUIRES internal token validation.
func (h *UserHandler) AdminResetAllUsers(ctx context.Context, req *connect.Request[userv1.AdminResetAllUsersRequest]) (*connect.Response[userv1.AdminResetAllUsersResponse], error) {
	return servicex.CallServiceWithToken(h.service.AdminResetAllUsers, h.logger, h.internalToken, "AdminResetAllUsers")(ctx, req)
}

// ValidateInternalToken handles ValidateInternalToken Connect requests.
// It validates an internal token by calling the greet service.
func (h *UserHandler) ValidateInternalToken(ctx context.Context, req *connect.Request[userv1.ValidateInternalTokenRequest]) (*connect.Response[userv1.ValidateInternalTokenResponse], error) {
	return servicex.CallService(h.service.ValidateInternalToken, h.logger, "ValidateInternalToken")(ctx, req)
}
