// Package internal provides internal implementation details for servicex.
package internal

import (
	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/connectx"
	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/obsx"
)

// BuildInterceptors creates the default Connect interceptors based on options.
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
