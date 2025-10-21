// Package servicex provides a unified microservice initialization framework.
package servicex

import (
	"connectrpc.com/connect"
	"github.com/eggybyte-technology/egg/connectx"
	"github.com/eggybyte-technology/egg/core/log"
	"github.com/eggybyte-technology/egg/obsx"
)

// buildInterceptors creates the default Connect interceptors based on options.
func buildInterceptors(opts *Options, logger log.Logger, otel *obsx.Provider) []connect.Interceptor {
	connectxOpts := connectx.Options{
		Logger:            logger,
		Otel:              otel,
		WithRequestBody:   opts.EnableDebugLogs,
		WithResponseBody:  opts.EnableDebugLogs,
		SlowRequestMillis: opts.SlowRequestMillis,
		PayloadAccounting: opts.PayloadAccounting,
	}

	return connectx.DefaultInterceptors(connectxOpts)
}
