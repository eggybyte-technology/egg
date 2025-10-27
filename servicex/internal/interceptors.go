// Package internal provides internal implementation details for servicex.
package internal

import (
	"connectrpc.com/connect"
	"go.eggybyte.com/egg/connectx"
	"go.eggybyte.com/egg/core/log"
	"go.eggybyte.com/egg/obsx"
)

// BuildInterceptors creates the default Connect interceptors based on options.
// enableDebugLogs determines whether to log request/response bodies (verbose logging).
func BuildInterceptors(logger log.Logger, otel *obsx.Provider, slowRequestMillis int64, enableDebugLogs, payloadAccounting bool) []connect.Interceptor {
	connectxOpts := connectx.Options{
		Logger:            logger,
		Otel:              otel,
		WithRequestBody:   enableDebugLogs,
		WithResponseBody:  enableDebugLogs,
		SlowRequestMillis: slowRequestMillis,
		PayloadAccounting: payloadAccounting,
	}

	return connectx.DefaultInterceptors(connectxOpts)
}
