package backend

import (
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/moduleinfo"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

func inventory(data []byte) ([]wasi.Import, uint64, error) { return moduleinfo.Inspect(data) }
