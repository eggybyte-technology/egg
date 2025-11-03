module go.eggybyte.com/egg/examples/connect-tester

go 1.25.1

replace go.eggybyte.com/egg/core => ../../core

replace go.eggybyte.com/egg/logx => ../../logx

replace go.eggybyte.com/egg/configx => ../../configx

replace go.eggybyte.com/egg/obsx => ../../obsx

replace go.eggybyte.com/egg/httpx => ../../httpx

replace go.eggybyte.com/egg/runtimex => ../../runtimex

replace go.eggybyte.com/egg/connectx => ../../connectx

replace go.eggybyte.com/egg/clientx => ../../clientx

replace go.eggybyte.com/egg/storex => ../../storex

replace go.eggybyte.com/egg/k8sx => ../../k8sx

replace go.eggybyte.com/egg/testingx => ../../testingx

replace go.eggybyte.com/egg/servicex => ../../servicex

replace go.eggybyte.com/egg/cli => ../../cli

replace go.eggybyte.com/egg/examples/minimal-connect-service => ../minimal-connect-service

replace go.eggybyte.com/egg/examples/user-service => ../user-service

require (
	connectrpc.com/connect v1.19.1
	go.eggybyte.com/egg/clientx v0.0.0-00010101000000-000000000000
	go.eggybyte.com/egg/core v0.0.0-00010101000000-000000000000
	go.eggybyte.com/egg/examples/minimal-connect-service v0.0.0-00010101000000-000000000000
	go.eggybyte.com/egg/examples/user-service v0.0.0-00010101000000-000000000000
	go.eggybyte.com/egg/logx v0.0.0-00010101000000-000000000000
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grafana/regexp v0.0.0-20240518133315-a468a5bfb3bc // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/otlptranslator v0.0.2 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	go.eggybyte.com/egg/obsx v0.0.0-00010101000000-000000000000 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.60.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk v1.38.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	golang.org/x/sys v0.36.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
)
