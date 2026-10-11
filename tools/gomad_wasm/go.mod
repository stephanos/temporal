module go.temporal.io/server/tools/gomad_wasm

go 1.27.1

require (
	go.temporal.io/server v0.0.0
	go.temporal.io/server/tools/gomad3 v0.0.0
)

require golang.org/x/mod v0.41.0 // indirect

replace go.temporal.io/server/tools/gomad3 => ../gomad3

replace go.temporal.io/server => ../..
