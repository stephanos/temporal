package measure

import (
	wasi "go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
)

func standaloneConfig() wasi.Config {
	return wasi.Config{Profile: wasi.StockProfile, Args: []string{"guest", "literal $arg"}, Environment: []string{"PWD=/workspace", "TZ=UTC", "ONLY=captured"}, WorkingDirectory: "/workspace", WritableDirectories: []string{"/workspace", "/tmp"}, EntropyKey: [32]byte{1, 2, 3}, Clock: wasi.ClockPolicy{EpochNanos: 946684800000000000, ReadStepNanos: 1000}, Limits: wasi.Limits{OutputBytes: 1 << 20, TranscriptBytes: 8 << 20, Calls: 10000, Descriptors: 64, Files: 1000, FilesystemBytes: 8 << 20, PendingEvents: 4096}}
}
