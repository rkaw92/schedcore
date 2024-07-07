module github.com/rkaw92/schedcore

go 1.22.0

replace github.com/gocql/gocql => github.com/scylladb/gocql v1.14.1

require (
	github.com/ThreeDotsLabs/watermill v1.3.5
	github.com/ThreeDotsLabs/watermill-amqp/v2 v2.1.2
	github.com/alexdrl/zerowater v0.0.3
	github.com/goccy/go-json v0.10.3
	github.com/gocql/gocql v1.6.0
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/robfig/cron v1.2.0
	github.com/rs/zerolog v1.33.0
)

require (
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/rabbitmq/amqp091-go v1.10.0 // indirect
)

require (
	github.com/danielgtaylor/huma/v2 v2.18.0
	github.com/go-chi/chi/v5 v5.1.0
	github.com/golang/snappy v0.0.4 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)
