package backend

import "go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"

func InspectModule(data []byte) ([]wasi.Import, uint64, error) { return inventory(data) }
