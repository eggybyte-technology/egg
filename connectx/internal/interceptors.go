// Package internal contains Connect interceptor implementations.
package internal

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/core/errors"
	"github.com/eggybyte-technology/egg/core/identity"
	"github.com/eggybyte-technology/egg/core/log"
	"gorm.io/gorm"
)

// RecoveryInterceptor creates a recovery interceptor that converts panics to errors.
func RecoveryInterceptor(logger log.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error(nil, "panic recovered", "panic", fmt.Sprintf("%v", r), "procedure", req.Spec().Procedure)
					err = connect.NewError(connect.CodeInternal, fmt.Errorf("internal server error: panic recovered"))
					resp = nil
				}
			}()
			return next(ctx, req)
		}
	}
}

// TimeoutInterceptor creates a timeout interceptor based on service-level configuration.
// Supports per-request timeout override via X-RPC-Timeout-Ms header (can only reduce, not increase).
func TimeoutInterceptor(defaultTimeoutMs int64) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			timeoutMs := defaultTimeoutMs

			// Check for request header override (can only reduce timeout)
			if req.Header() != nil {
				if headerTimeout := req.Header().Get("X-RPC-Timeout-Ms"); headerTimeout != "" {
					if parsed, err := strconv.ParseInt(headerTimeout, 10, 64); err == nil {
						if parsed > 0 && parsed < timeoutMs {
							timeoutMs = parsed
						}
					}
				}
			}

			// Apply timeout if configured
			if timeoutMs > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
				defer cancel()
			}

			return next(ctx, req)
		}
	}
}

// LoggingInterceptor creates a logging interceptor for structured request/response logging.
func LoggingInterceptor(logger log.Logger, opts LoggingOptions) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			startTime := time.Now()

			// Extract request context for logging
			var requestContext []any

			// Extract user information from context if available
			if userInfo, ok := identity.UserFrom(ctx); ok {
				requestContext = append(requestContext, log.Str("user_id", userInfo.UserID))
			}

			// Extract request metadata if available
			if requestMeta, ok := identity.MetaFrom(ctx); ok {
				if requestMeta.RequestID != "" {
					requestContext = append(requestContext, log.Str("request_id", requestMeta.RequestID))
				}
			}

			// Log request started
			fields := append([]any{log.Str("procedure", req.Spec().Procedure)}, requestContext...)
			logger.Info("request started", fields...)

			// Call next handler
			resp, err := next(ctx, req)

			// Log response
			duration := time.Since(startTime)
			fields = append([]any{
				log.Str("procedure", req.Spec().Procedure),
				log.Dur("duration", duration),
			}, requestContext...)

			if err != nil {
				// Only log as ERROR if it's a real server error, not business logic errors
				// Business logic errors (like not found, already exists) should be logged as INFO
				if isServerError(err) {
					fields = append(fields, log.Str("error_type", "server_error"))
					logger.Error(err, "request failed", fields...)
				} else {
					fields = append(fields, log.Str("status", "business_error"))
					logger.Info("request completed", fields...)
				}
			} else {
				fields = append(fields, log.Str("status", "success"))
				logger.Info("request completed", fields...)
			}

			return resp, err
		}
	}
}

// IdentityInterceptor creates an identity injection interceptor.
func IdentityInterceptor(headers HeaderMapping) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Extract identity from headers using Connect's Header() method
			if req.Header() != nil {
				userInfo, requestMeta := extractIdentityFromConnectHeaders(req.Header(), headers)

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
				connectCode := mapErrorToConnectCode(err)
				return resp, connect.NewError(connectCode, err)
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

// extractIdentityFromConnectHeaders extracts user identity and request metadata from Connect headers.
func extractIdentityFromConnectHeaders(headers http.Header, mapping HeaderMapping) (*identity.UserInfo, *identity.RequestMeta) {
	var userInfo *identity.UserInfo
	var requestMeta *identity.RequestMeta

	// Extract user information
	if userID := headers.Get(mapping.UserID); userID != "" {
		userInfo = &identity.UserInfo{
			UserID:   userID,
			UserName: headers.Get(mapping.UserName),
		}

		// Parse roles
		if rolesHeader := headers.Get(mapping.Roles); rolesHeader != "" {
			userInfo.Roles = strings.Split(rolesHeader, ",")
			for i, role := range userInfo.Roles {
				userInfo.Roles[i] = strings.TrimSpace(role)
			}
		}
	}

	// Extract request metadata
	requestMeta = &identity.RequestMeta{
		RequestID:     headers.Get(mapping.RequestID),
		InternalToken: headers.Get(mapping.InternalToken),
		UserAgent:     headers.Get(mapping.UserAgent),
	}

	// Determine remote IP
	if realIP := headers.Get(mapping.RealIP); realIP != "" {
		requestMeta.RemoteIP = realIP
	} else if forwardedFor := headers.Get(mapping.ForwardedFor); forwardedFor != "" {
		// Take the first IP from X-Forwarded-For
		if firstIP := strings.Split(forwardedFor, ",")[0]; firstIP != "" {
			requestMeta.RemoteIP = strings.TrimSpace(firstIP)
		}
	}

	return userInfo, requestMeta
}

// mapErrorToConnectCode maps core/errors.Code to Connect error codes.
func mapErrorToConnectCode(err error) connect.Code {
	code := errors.CodeOf(err)
	switch code {
	case errors.CodeInvalidArgument:
		return connect.CodeInvalidArgument
	case errors.CodeNotFound:
		return connect.CodeNotFound
	case errors.CodeAlreadyExists:
		return connect.CodeAlreadyExists
	case errors.CodePermissionDenied:
		return connect.CodePermissionDenied
	case errors.CodeUnauthenticated:
		return connect.CodeUnauthenticated
	case errors.CodeResourceExhausted:
		return connect.CodeResourceExhausted
	case errors.CodeInternal:
		return connect.CodeInternal
	case errors.CodeUnavailable:
		return connect.CodeUnavailable
	case errors.CodeDeadlineExceeded:
		return connect.CodeDeadlineExceeded
	case errors.CodeUnimplemented:
		return connect.CodeUnimplemented
	case errors.CodeAborted:
		return connect.CodeAborted
	case errors.CodeOutOfRange:
		return connect.CodeOutOfRange
	case errors.CodeDataLoss:
		return connect.CodeDataLoss
	default:
		return connect.CodeInternal
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
	}

	return fields
}

// isServerError determines if an error is a real server error that should be logged as ERROR,
// or a business logic error that should be logged as INFO.
// Business logic errors (like not found, already exists) are expected and should not be ERROR level.
func isServerError(err error) bool {
	if err == nil {
		return false
	}

	// Check for database connection errors (real server errors)
	if err == sql.ErrConnDone ||
		err == sql.ErrTxDone ||
		strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "network is unreachable") {
		return true
	}

	// Check for GORM connection errors
	if err == gorm.ErrRecordNotFound {
		return false // This is a business logic error, not a server error
	}

	// Check core error codes - only internal errors should be treated as server errors
	errorCode := errors.CodeOf(err)
	switch errorCode {
	case errors.CodeInvalidArgument,
		errors.CodeNotFound,
		errors.CodeAlreadyExists,
		errors.CodePermissionDenied,
		errors.CodeUnauthenticated,
		errors.CodeResourceExhausted,
		errors.CodeDeadlineExceeded,
		errors.CodeUnimplemented,
		errors.CodeAborted,
		errors.CodeOutOfRange,
		errors.CodeDataLoss:
		return false // These are business logic errors
	case errors.CodeInternal, errors.CodeUnavailable:
		return true // These are real server errors
	default:
		// Unknown error code, treat as server error for safety
		return true
	}
}
