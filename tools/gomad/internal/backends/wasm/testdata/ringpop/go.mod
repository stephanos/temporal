module gomad.test/ringpop

go 1.27.1

require (
	github.com/temporalio/ringpop-go v0.1.0
	github.com/temporalio/tchannel-go v1.22.1
	go.temporal.io/server v0.0.0
)

require (
	github.com/apache/thrift v0.24.0 // indirect
	github.com/benbjohnson/clock v1.3.5 // indirect
	github.com/cactus/go-statsd-client/statsd v0.0.0-20200423205355-cb0885a1018c // indirect
	github.com/dgryski/go-farm v0.0.0-20240924180020-3414d57e47da // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/uber-common/bark v1.3.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.58.0 // indirect
)

// The server module pins every transitive version to the one Temporal builds with.
replace go.temporal.io/server => ../../../../../../..
