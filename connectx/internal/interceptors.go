// Package internal contains Connect interceptor implementations.
package internal

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/core/errors"
	"github.com/eggybyte-technology/egg/core/identity"
	"github.com/eggybyte-technology/egg/core/log"
)

// RecoveryInterceptor creates a recovery interceptor that converts panics to errors.
func RecoveryInterceptor(logger log.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error(nil, "panic recovered", log.Str("panic", fmt.Sprintf("%v", r)))
				}
			}()
			return next(ctx, req)
		}
	}
}

// LoggingInterceptor creates a logging interceptor for structured request/response logging.
func LoggingInterceptor(logger log.Logger, opts LoggingOptions) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			startTime := time.Now()

			// Log request
			logger.Info("request started",
				log.Str("procedure", req.Spec().Procedure),
			)

			// Call next handler
			resp, err := next(ctx, req)

			// Log response
			duration := time.Since(startTime)
			if err != nil {
				logger.Error(err, "request failed",
					log.Str("procedure", req.Spec().Procedure),
					log.Dur("duration", duration),
				)
			} else {
				logger.Info("request completed",
					log.Str("procedure", req.Spec().Procedure),
					log.Dur("duration", duration),
				)
			}

			return resp, err
		}
	}
}

// IdentityInterceptor creates an identity injection interceptor.
func IdentityInterceptor(headers HeaderMapping) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Extract identity from headers
			if httpReq := req.Any().(*http.Request); httpReq != nil {
				userInfo, requestMeta := extractIdentityFromHeaders(httpReq, headers)

				// Inject into context
				if userInfo != nil {
					ctx = identity.WithUser(ctx, userInfo)
				}
				if requestMeta != nil {
					ctx = identity.WithMeta(ctx, requestMeta)
				}
			}

			return next(ctx, req)
		}
	}
}

// ErrorMappingInterceptor creates an error mapping interceptor.
func ErrorMappingInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			resp, err := next(ctx, req)
			if err != nil {
				// Map core/errors to Connect codes
				_ = mapErrorToConnectCode(err)
				return resp, connect.NewError(connect.CodeUnknown, err)
			}
			return resp, err
		}
	}
}

// LoggingOptions holds configuration for the logging interceptor.
type LoggingOptions struct {
	WithRequestBody   bool
	WithResponseBody  bool
	SlowRequestMillis int64
	PayloadAccounting bool
}

// HeaderMapping defines header to field mapping.
type HeaderMapping struct {
	RequestID     string
	InternalToken string
	UserID        string
	UserName      string
	Tenant        string
	Roles         string
	RealIP        string
	ForwardedFor  string
	UserAgent     string
}

// extractIdentityFromHeaders extracts user identity and request metadata from HTTP headers.
func extractIdentityFromHeaders(req *http.Request, headers HeaderMapping) (*identity.UserInfo, *identity.RequestMeta) {
	var userInfo *identity.UserInfo
	var requestMeta *identity.RequestMeta

	// Extract user information
	if userID := req.Header.Get(headers.UserID); userID != "" {
		userInfo = &identity.UserInfo{
			UserID:   userID,
			UserName: req.Header.Get(headers.UserName),
			Tenant:   req.Header.Get(headers.Tenant),
		}

		// Parse roles
		if rolesHeader := req.Header.Get(headers.Roles); rolesHeader != "" {
			userInfo.Roles = strings.Split(rolesHeader, ",")
			for i, role := range userInfo.Roles {
				userInfo.Roles[i] = strings.TrimSpace(role)
			}
		}
	}

	// Extract request metadata
	requestMeta = &identity.RequestMeta{
		RequestID:     req.Header.Get(headers.RequestID),
		InternalToken: req.Header.Get(headers.InternalToken),
		UserAgent:     req.Header.Get(headers.UserAgent),
	}

	// Determine remote IP
	if realIP := req.Header.Get(headers.RealIP); realIP != "" {
		requestMeta.RemoteIP = realIP
	} else if forwardedFor := req.Header.Get(headers.ForwardedFor); forwardedFor != "" {
		// Take the first IP from X-Forwarded-For
		if firstIP := strings.Split(forwardedFor, ",")[0]; firstIP != "" {
			requestMeta.RemoteIP = strings.TrimSpace(firstIP)
		}
	} else {
		requestMeta.RemoteIP = req.RemoteAddr
	}

	return userInfo, requestMeta
}

// mapErrorToConnectCode maps core/errors.Code to Connect error codes.
func mapErrorToConnectCode(err error) string {
	code := errors.CodeOf(err)
	switch code {
	case errors.CodeInvalidArgument:
		return "invalid_argument"
	case errors.CodeNotFound:
		return "not_found"
	case errors.CodeAlreadyExists:
		return "already_exists"
	case errors.CodePermissionDenied:
		return "permission_denied"
	case errors.CodeUnauthenticated:
		return "unauthenticated"
	case errors.CodeInternal:
		return "internal"
	case errors.CodeUnavailable:
		return "unavailable"
	case errors.CodeDeadlineExceeded:
		return "deadline_exceeded"
	default:
		return "internal"
	}
}

// logRequestFields creates structured log fields for request logging.
func logRequestFields(req *http.Request, startTime time.Time, userInfo *identity.UserInfo, requestMeta *identity.RequestMeta) []any {
	fields := []any{
		log.Str("method", req.Method),
		log.Str("path", req.URL.Path),
		log.Str("remote_ip", requestMeta.RemoteIP),
		log.Str("user_agent", requestMeta.UserAgent),
		log.Dur("latency_ms", time.Since(startTime)),
	}

	if requestMeta.RequestID != "" {
		fields = append(fields, log.Str("request_id", requestMeta.RequestID))
	}

	if userInfo != nil {
		fields = append(fields, log.Str("user_id", userInfo.UserID))
		if userInfo.Tenant != "" {
			fields = append(fields, log.Str("tenant", userInfo.Tenant))
		}
	}

	return fields
}
