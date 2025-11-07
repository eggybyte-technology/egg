// Package internal provides handler helpers for reducing boilerplate in Connect handlers.
//
// This file contains helper functions that simplify common patterns in Connect RPC handlers,
// such as automatic debug logging, error handling, and internal token validation.

package internal

import (
	"context"

	"connectrpc.com/connect"
	"go.eggybyte.com/egg/core/identity"
	"go.eggybyte.com/egg/core/log"
)

// HandlerFunc is a generic handler function signature for Connect RPC methods.
//
// This type represents a handler function that takes a context and request,
// and returns a response or error. It's used by CallService to provide
// automatic logging and error handling.
type HandlerFunc[TReq, TResp any] func(ctx context.Context, req *connect.Request[TReq]) (*connect.Response[TResp], error)

// ServiceCaller represents a service method that takes a proto request and returns a proto response.
type ServiceCaller[TReq, TResp any] func(ctx context.Context, req *TReq) (*TResp, error)

// CallService wraps a service method call with automatic error handling and Connect response conversion.
//
// This is the simplest way to create Connect handlers from service methods.
// It automatically handles:
// - Converting Connect request to proto message (req.Msg)
// - Calling the service method
// - Converting proto response to Connect response
// - Error handling
// - Debug logging
//
// Usage:
//
//	func (h *UserHandler) CreateUser(ctx context.Context, req *connect.Request[userv1.CreateUserRequest]) (*connect.Response[userv1.CreateUserResponse], error) {
//	    return CallService(h.service.CreateUser, h.logger, "CreateUser")(ctx, req)
//	}
func CallService[TReq, TResp any](serviceMethod ServiceCaller[TReq, TResp], logger log.Logger, methodName string) HandlerFunc[TReq, TResp] {
	return func(ctx context.Context, req *connect.Request[TReq]) (*connect.Response[TResp], error) {
		// Log incoming request at debug level (Connect interceptor handles request lifecycle logging)
		logger.Debug("handler processing request",
			log.Str("method", methodName))

		// Call service method with proto message
		resp, err := serviceMethod(ctx, req.Msg)
		if err != nil {
			// Error logging is handled by Connect interceptor with proper error classification
			return nil, err
		}

		// Success logging is handled by Connect interceptor
		logger.Debug("handler completed successfully",
			log.Str("method", methodName))

		return connect.NewResponse(resp), nil
	}
}

// WithInternalToken wraps a handler function with internal token validation.
func WithInternalToken[TReq, TResp any](fn HandlerFunc[TReq, TResp], internalToken string, logger log.Logger, methodName string) HandlerFunc[TReq, TResp] {
	return func(ctx context.Context, req *connect.Request[TReq]) (*connect.Response[TResp], error) {
		// Validate internal token before executing handler
		if err := identity.RequireInternalToken(ctx, internalToken); err != nil {
			logger.Warn("handler rejected: invalid internal token",
				log.Str("method", methodName))
			return nil, err
		}

		logger.Info("handler authorized, proceeding",
			log.Str("method", methodName))

		// Execute wrapped handler
		return fn(ctx, req)
	}
}

// CallServiceWithToken wraps a service method call with token validation.
//
// This combines CallService with internal token validation for admin operations.
// It validates the token before calling the service method.
func CallServiceWithToken[TReq, TResp any](serviceMethod ServiceCaller[TReq, TResp], logger log.Logger, internalToken string, methodName string) HandlerFunc[TReq, TResp] {
	return WithInternalToken(CallService(serviceMethod, logger, methodName), internalToken, logger, methodName)
}
