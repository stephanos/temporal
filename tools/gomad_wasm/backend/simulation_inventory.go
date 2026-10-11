package backend

import "go.temporal.io/server/tools/gomad_wasm/wasi"

func InspectModule(data []byte) ([]wasi.Import, uint64, error) { return inventory(data) }
