module umpire.local/temporal

go 1.27.0

require (
	github.com/tetratelabs/wazero v1.12.0
	go.temporal.io/api v1.63.6-0.20260909222256-20151aa90480
	google.golang.org/grpc v1.83.1
	google.golang.org/protobuf v1.36.11
	umpire.local/model v0.0.0
)

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0 // indirect
	github.com/nexus-rpc/nexus-proto-annotations v0.1.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
)

replace umpire.local/model => ../../../umpire/hosts/go
