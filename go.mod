module github.com/rkaw92/schedcore

go 1.22.0

replace github.com/gocql/gocql => github.com/scylladb/gocql v1.14.1

require (
	github.com/gocql/gocql v1.6.0
	github.com/google/uuid v1.6.0
)

require (
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.5.4 // indirect
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)