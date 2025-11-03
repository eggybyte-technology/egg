module go.eggybyte.com/egg/cli

go 1.25.1

replace go.eggybyte.com/egg/core => ../core

replace go.eggybyte.com/egg/logx => ../logx

replace go.eggybyte.com/egg/configx => ../configx

replace go.eggybyte.com/egg/obsx => ../obsx

replace go.eggybyte.com/egg/httpx => ../httpx

replace go.eggybyte.com/egg/runtimex => ../runtimex

replace go.eggybyte.com/egg/connectx => ../connectx

replace go.eggybyte.com/egg/clientx => ../clientx

replace go.eggybyte.com/egg/storex => ../storex

replace go.eggybyte.com/egg/k8sx => ../k8sx

replace go.eggybyte.com/egg/testingx => ../testingx

replace go.eggybyte.com/egg/servicex => ../servicex

require (
	github.com/spf13/cobra v1.10.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
)
