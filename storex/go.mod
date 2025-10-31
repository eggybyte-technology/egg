module go.eggybyte.com/egg/storex

go 1.25.1

replace go.eggybyte.com/egg/core => /Users/fengguangyao/eggybyte/projects/go/egg/core

replace go.eggybyte.com/egg/logx => /Users/fengguangyao/eggybyte/projects/go/egg/logx

replace go.eggybyte.com/egg/configx => /Users/fengguangyao/eggybyte/projects/go/egg/configx

replace go.eggybyte.com/egg/obsx => /Users/fengguangyao/eggybyte/projects/go/egg/obsx

replace go.eggybyte.com/egg/httpx => /Users/fengguangyao/eggybyte/projects/go/egg/httpx

replace go.eggybyte.com/egg/runtimex => /Users/fengguangyao/eggybyte/projects/go/egg/runtimex

replace go.eggybyte.com/egg/connectx => /Users/fengguangyao/eggybyte/projects/go/egg/connectx

replace go.eggybyte.com/egg/clientx => /Users/fengguangyao/eggybyte/projects/go/egg/clientx

require (
	go.eggybyte.com/egg/core v0.3.1
	gorm.io/driver/mysql v1.6.0
	gorm.io/driver/postgres v1.6.0
	gorm.io/driver/sqlite v1.6.0
	gorm.io/gorm v1.31.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/crypto v0.42.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)
