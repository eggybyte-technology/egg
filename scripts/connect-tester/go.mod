module github.com/eggybyte-technology/egg/scripts/connect-tester

go 1.25.1

require (
	connectrpc.com/connect v1.19.1
	github.com/eggybyte-technology/egg/examples/minimal-connect-service v0.0.0
	github.com/eggybyte-technology/egg/examples/user-service v0.0.0-00010101000000-000000000000
)

require (
	github.com/google/go-cmp v0.6.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
)

replace github.com/eggybyte-technology/egg/examples/minimal-connect-service => ../../examples/minimal-connect-service

replace github.com/eggybyte-technology/egg/examples/user-service => ../../examples/user-service
