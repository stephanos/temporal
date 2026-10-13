package backend

import "go.temporal.io/server/tools/gomad3/internal/backends/wasm/wasi"

func InspectModule(data []byte) ([]wasi.Import, uint64, error) { return inventory(data) }
