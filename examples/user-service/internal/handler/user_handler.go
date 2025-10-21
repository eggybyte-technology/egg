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
	"github.com/eggybyte-technology/egg/core/log"
	userv1 "github.com/eggybyte-technology/egg/examples/user-service/gen/go/user/v1"
	userv1connect "github.com/eggybyte-technology/egg/examples/user-service/gen/go/user/v1/userv1connect"
	"github.com/eggybyte-technology/egg/examples/user-service/internal/service"
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
func NewUserHandler(service service.UserService, logger log.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  logger,
	}
}

// CreateUser handles CreateUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) CreateUser(ctx context.Context, req *connect.Request[userv1.CreateUserRequest]) (*connect.Response[userv1.CreateUserResponse], error) {
	h.logger.Info("CreateUser request received",
		log.Str("email", req.Msg.Email),
		log.Str("name", req.Msg.Name))

	// Delegate to business service
	response, err := h.service.CreateUser(ctx, req.Msg)
	if err != nil {
		h.logger.Error(err, "CreateUser failed",
			log.Str("email", req.Msg.Email))
		return nil, err
	}

	h.logger.Info("CreateUser completed successfully",
		log.Str("user_id", response.User.Id))

	return connect.NewResponse(response), nil
}

// GetUser handles GetUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) GetUser(ctx context.Context, req *connect.Request[userv1.GetUserRequest]) (*connect.Response[userv1.GetUserResponse], error) {
	h.logger.Info("GetUser request received",
		log.Str("user_id", req.Msg.Id))

	// Delegate to business service
	response, err := h.service.GetUser(ctx, req.Msg)
	if err != nil {
		h.logger.Error(err, "GetUser failed",
			log.Str("user_id", req.Msg.Id))
		return nil, err
	}

	h.logger.Info("GetUser completed successfully",
		log.Str("user_id", response.User.Id))

	return connect.NewResponse(response), nil
}

// UpdateUser handles UpdateUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) UpdateUser(ctx context.Context, req *connect.Request[userv1.UpdateUserRequest]) (*connect.Response[userv1.UpdateUserResponse], error) {
	h.logger.Info("UpdateUser request received",
		log.Str("user_id", req.Msg.Id),
		log.Str("email", req.Msg.Email))

	// Delegate to business service
	response, err := h.service.UpdateUser(ctx, req.Msg)
	if err != nil {
		h.logger.Error(err, "UpdateUser failed",
			log.Str("user_id", req.Msg.Id))
		return nil, err
	}

	h.logger.Info("UpdateUser completed successfully",
		log.Str("user_id", response.User.Id))

	return connect.NewResponse(response), nil
}

// DeleteUser handles DeleteUser Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) DeleteUser(ctx context.Context, req *connect.Request[userv1.DeleteUserRequest]) (*connect.Response[userv1.DeleteUserResponse], error) {
	h.logger.Info("DeleteUser request received",
		log.Str("user_id", req.Msg.Id))

	// Delegate to business service
	response, err := h.service.DeleteUser(ctx, req.Msg)
	if err != nil {
		h.logger.Error(err, "DeleteUser failed",
			log.Str("user_id", req.Msg.Id))
		return nil, err
	}

	h.logger.Info("DeleteUser completed successfully",
		log.Str("user_id", req.Msg.Id),
		log.Str("success", "true"))

	return connect.NewResponse(response), nil
}

// ListUsers handles ListUsers Connect requests.
// It validates the request and delegates to the business service.
func (h *UserHandler) ListUsers(ctx context.Context, req *connect.Request[userv1.ListUsersRequest]) (*connect.Response[userv1.ListUsersResponse], error) {
	h.logger.Info("ListUsers request received",
		log.Int("page", int(req.Msg.Page)),
		log.Int("page_size", int(req.Msg.PageSize)))

	// Delegate to business service
	response, err := h.service.ListUsers(ctx, req.Msg)
	if err != nil {
		h.logger.Error(err, "ListUsers failed",
			log.Int("page", int(req.Msg.Page)))
		return nil, err
	}

	h.logger.Info("ListUsers completed successfully",
		log.Int("count", len(response.Users)),
		log.Int("total", int(response.Total)))

	return connect.NewResponse(response), nil
}
